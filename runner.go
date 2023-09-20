package loaderbot

import (
	"context"
	"encoding/csv"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"runtime"
	"strconv"
	"sync"
	"sync/atomic"
	"time"

	"github.com/google/gops/agent"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"go.uber.org/ratelimit"
)

const (
	DefaultResultsQueueCapacity = 100_000
	MetricsLogFile              = "requests_%s_%s_%d.csv"
	PercsLogFile                = "percs_%s_%s_%d.csv"
	ReportGraphFile             = "percs_%s_%s_%d.html"
	BoundRPSTickTemplate        = "step: %d, tick: %d, attackers: [%d], rate [%.4f -> %v], perc: 50 [%v] 95 [%v] 99 [%v], # requests [%d], %% success [%.4f]"
	UnboundRPSTickTemplate      = "attackers: [%d], rate [%.4f], perc: 50 [%v] 95 [%v] 99 [%v], # requests [%d], %% success [%.4f]"
)

var (
	promOnce         = &sync.Once{}
	ResultsCsvHeader = []string{"RequestLabel", "BeginTimeNano", "EndTimeNano", "Elapsed", "StatusCode", "Error"}
	PercsCsvHeader   = []string{"RequestLabel", "Tick", "RPS", "P50", "P95", "P99"}
)

// Controlled struct for adding test vars
type Controlled struct {
	Sleep int64
}

// TestData shared test data
type TestData struct {
	*sync.Mutex
	Index int
	Data  interface{}
}

type ClusterTickMetrics struct {
	Samples [][]AttackResult
	Metrics *Metrics
}

type TickMetrics struct {
	Samples  []AttackResult
	Metrics  *Metrics
	Reported bool
}

type attackToken struct {
	TargetRPS int
	Step      int
	Tick      int
}

func (a attackToken) String() string {
	return fmt.Sprintf("targetRPS: %d, step: %d, tick: %d", a.TargetRPS, a.Step, a.Tick)
}

// Runner provides test context for attacking target with constant amount of runners with a schedule
type Runner struct {
	// Name of a runner
	Name string
	// Cfg runner config
	Cfg *RunnerConfig
	// prototype from which all attackers cloned
	attackerPrototype Attack
	// target RPS for step, changed every step
	targetRPS int
	// metrics for every received tick (completed requests)
	receivedTickMetricsMu *sync.Mutex
	receivedTickMetrics   map[int]*TickMetrics
	// ratelimiter for keeping constant rps inside test step
	rl ratelimit.Limiter
	// TimeoutCtx test timeout ctx
	TimeoutCtx context.Context
	// test cancel func
	CancelFunc context.CancelFunc
	// next schedule chan to signal to attack
	next chan attackToken
	// attackers cloned for a prototype
	attackers []Attack

	// inner Results chan, when used in standalone mode
	results chan AttackResult
	// outer Results chan, when called as a service, sends Results in batches
	OutResults chan []AttackResult
	// uniq error messages
	uniqErrors map[string]int
	// Failed means there some errors in test
	Failed int64
	// Report data
	Report *Report
	// data used to control attackers in test
	controlled Controlled
	// TestData data shared between attackers during test
	TestData       interface{}
	HTTPClient     *http.Client
	FastHTTPClient *FastHTTPClient
	PromReporter   *PromReporter
	wg             *sync.WaitGroup
	L              *Logger
}

// NewRunner creates new runner with constant amount of attackers by RunnerConfig
func NewRunner(cfg *RunnerConfig, a Attack, data interface{}) *Runner {
	cfg.Validate()
	cfg.DefaultCfgValues()
	var rl ratelimit.Limiter
	if cfg.SystemMode == BoundRPS || cfg.SystemMode == BoundRPSAutoscale {
		rl = ratelimit.New(cfg.StartRPS)
	}
	r := &Runner{
		Name:                  cfg.Name,
		Cfg:                   cfg,
		attackerPrototype:     a,
		targetRPS:             cfg.StartRPS,
		next:                  make(chan attackToken),
		rl:                    rl,
		attackers:             make([]Attack, 0),
		results:               make(chan AttackResult, DefaultResultsQueueCapacity),
		OutResults:            make(chan []AttackResult, DefaultResultsQueueCapacity),
		receivedTickMetricsMu: &sync.Mutex{},
		receivedTickMetrics:   make(map[int]*TickMetrics),
		uniqErrors:            make(map[string]int),
		controlled:            Controlled{},
		TestData:              data,
		HTTPClient:            NewLoggingHTTPClient(cfg.DumpTransport, cfg.AttackerTimeout),
		FastHTTPClient:        NewLoggingFastHTTPClient(cfg.DumpTransport),
		wg:                    &sync.WaitGroup{},
		L:                     NewLogger(cfg).With("runner", cfg.Name),
	}
	for i := 0; i < cfg.Attackers; i++ {
		a := r.attackerPrototype.Clone(r)
		if err := a.Setup(*r.Cfg); err != nil {
			log.Fatal(errAttackerSetup)
		}
		r.attackers = append(r.attackers, a)
	}
	if cfg.ReportOptions.CSV {
		r.Report = NewReport(r.Cfg)
	}
	if cfg.Prometheus != nil && cfg.Prometheus.Enable {
		r.PromReporter = NewPromReporter(r.Name)
		promOnce.Do(func() {
			go func() {
				http.Handle("/metrics", promhttp.Handler())
				if err := http.ListenAndServe(fmt.Sprintf(":%d", cfg.Prometheus.Port), nil); err != nil {
					r.L.Error(err)
				}
			}()
		})
	}
	return r
}

// Run runs the test
func (r *Runner) Run(serverCtx context.Context) (float64, error) {
	_ = agent.Listen(agent.Options{
		Addr: "0.0.0.0:10500",
	})

	runStartTime := time.Now()
	if r.Cfg.WaitBeforeSec > 0 {
		r.L.Infof("waiting for %d seconds before start", r.Cfg.WaitBeforeSec)
		time.Sleep(time.Duration(r.Cfg.WaitBeforeSec) * time.Second)
	}
	r.L.Infof("runner started, mode: %s", r.Cfg.SystemMode.String())
	if serverCtx == nil {
		serverCtx = context.Background()
	}
	r.TimeoutCtx, r.CancelFunc = context.WithTimeout(serverCtx, time.Duration(r.Cfg.TestTimeSec)*time.Second)
	for atkIdx, attacker := range r.attackers {
		r.L.Debugf("starting attacker: %d", atkIdx)
		go attack(attacker, r)
	}
	r.handleShutdownSignal()
	r.schedule()
	r.collectResults()
	<-r.TimeoutCtx.Done()
	r.wg.Wait()
	r.L.Infof("shutting down")
	r.L.Infof("total run time: %.2f sec", time.Since(runStartTime).Seconds())
	var maxRPS float64
	if r.Cfg.ReportOptions.CSV {
		r.Report.flushLogs()
		r.Report.plot()
		maxRPS = r.maxRPS()
		r.L.Infof("max rps: %.2f", maxRPS)
	}
	r.safeCloseIdleConnections()
	r.L.Infof("runner exited")
	return maxRPS, nil
}

func (r *Runner) safeCloseIdleConnections() {
	defer func() {
		if rec := recover(); rec != nil {
			// we don't know if user will use default http client,
			// client panics if there is no connections but we closing them
			// https://golang.org/src/net/http/transport.go#L1447
			r.L.Infof("no http connections were made, safe closing connections")
		}
	}()
	r.HTTPClient.CloseIdleConnections()
}

// schedule creates schedule plan for a test
func (r *Runner) schedule() {
	go func() {
		var (
			currentStep         = 1
			currentTick         = 1
			ticksInStep         = r.Cfg.StepDurationSec
			totalRequestsFired  = 0
			requestsFiredInTick = 0
		)
		if r.Cfg.SystemMode == UnboundRPS {
			// analyze 100 samples by each attacker if no rps requirements
			r.targetRPS = len(r.attackers) * 100
			ticksInStep = 1
		}
		for {
			select {
			case <-r.TimeoutCtx.Done():
				r.L.Infof("total requests fired: %d", totalRequestsFired)
				close(r.next)
				return
			default:
				if r.rl != nil {
					r.rl.Take()
				}
				// either schedule attack and count requests, or retry in limiter pace
				select {
				case r.next <- attackToken{
					TargetRPS: r.targetRPS,
					Step:      currentStep,
					Tick:      currentTick,
				}:
				default:
					continue
				}
				totalRequestsFired++
				requestsFiredInTick++
				if requestsFiredInTick == r.targetRPS {
					currentTick += 1
					requestsFiredInTick = 0
					r.L.Infof("active goroutines: %d", runtime.NumGoroutine())
					if currentTick%ticksInStep == 0 {
						if r.rl != nil {
							r.targetRPS += r.Cfg.StepRPS
							r.rl = ratelimit.New(r.targetRPS)
							currentStep += 1
							r.L.Infof("next step: step -> %d, rps -> %d", currentStep, r.targetRPS)
						}
					}
				}
			}
		}
	}()
}

// collectResults collects attackers Results and writes them to one of report options
func (r *Runner) collectResults() {
	r.wg.Add(1)
	go func() {
		defer r.wg.Done()
		var (
			totalRequestsStored = 0
		)
		for {
			select {
			case <-r.TimeoutCtx.Done():
				r.L.Infof("total requests stored: %d", totalRequestsStored)
				r.printErrors()
				close(r.OutResults)
				return
			case res := <-r.results:
				r.L.Debugf("received result: %v", res)
				totalRequestsStored++

				errorForReport := "ok"
				if res.DoResult.Error != "" {
					r.uniqErrors[res.DoResult.Error] += 1
					r.L.Debugf("attacker error: %s", res.DoResult.Error)
					errorForReport = res.DoResult.Error
				}

				if r.Cfg.ReportOptions.CSV {
					r.Report.writeResultEntry(res, errorForReport)
				}
				r.processTickMetrics(res)
			}
		}
	}()
}

// scaleAttackers scaling attackers to meet targetRPS
func (r *Runner) scaleAttackers(tm *TickMetrics) {
	if r.Cfg.SystemMode == BoundRPSAutoscale && tm.Metrics.Rate < float64(tm.Samples[0].AttackToken.TargetRPS)*r.Cfg.AttackersScaleThreshold {
		r.L.Infof("scaling attackers: %d", r.Cfg.AttackersScaleAmount)
		for i := 0; i < r.Cfg.AttackersScaleAmount; i++ {
			a := r.attackerPrototype.Clone(r)
			if err := a.Setup(*r.Cfg); err != nil {
				log.Fatal(errAttackerSetup)
			}
			r.attackers = append(r.attackers, a)
			go attack(a, r)
		}
	}
}

// processTickMetrics add attack result to tick metrics, if it's last result in tick then report
func (r *Runner) processTickMetrics(res AttackResult) {
	// if no such tick, create new TickMetrics
	r.receivedTickMetricsMu.Lock()
	defer r.receivedTickMetricsMu.Unlock()
	if _, ok := r.receivedTickMetrics[res.AttackToken.Tick]; !ok {
		r.receivedTickMetrics[res.AttackToken.Tick] = &TickMetrics{
			make([]AttackResult, 0),
			NewMetrics(),
			false,
		}
	}
	currentTickMetrics := r.receivedTickMetrics[res.AttackToken.Tick]
	currentTickMetrics.Samples = append(currentTickMetrics.Samples, res)
	if len(currentTickMetrics.Samples) == res.AttackToken.TargetRPS && !currentTickMetrics.Reported {
		if r.Cfg.ReportOptions.Stream {
			r.OutResults <- currentTickMetrics.Samples
		}
		for _, s := range currentTickMetrics.Samples {
			currentTickMetrics.Metrics.add(s)
		}
		currentTickMetrics.Metrics.update()
		if currentTickMetrics.Metrics.Success < r.Cfg.SuccessRatio {
			r.L.Infof("success ratio threshold reached: %.4f < %.4f", currentTickMetrics.Metrics.Success, r.Cfg.SuccessRatio)
			atomic.AddInt64(&r.Failed, 1)
			r.CancelFunc()
		}
		switch r.Cfg.SystemMode {
		case BoundRPS:
			fallthrough
		case BoundRPSAutoscale:
			r.L.Infof(
				BoundRPSTickTemplate,
				res.AttackToken.Step,
				res.AttackToken.Tick,
				len(r.attackers),
				currentTickMetrics.Metrics.Rate,
				res.AttackToken.TargetRPS,
				currentTickMetrics.Metrics.Latencies.P50,
				currentTickMetrics.Metrics.Latencies.P95,
				currentTickMetrics.Metrics.Latencies.P99,
				currentTickMetrics.Metrics.Requests,
				currentTickMetrics.Metrics.successLogEntry(),
			)
		case UnboundRPS:
			r.L.Infof(
				UnboundRPSTickTemplate,
				len(r.attackers),
				currentTickMetrics.Metrics.Rate,
				currentTickMetrics.Metrics.Latencies.P50,
				currentTickMetrics.Metrics.Latencies.P95,
				currentTickMetrics.Metrics.Latencies.P99,
				currentTickMetrics.Metrics.Requests,
				currentTickMetrics.Metrics.successLogEntry(),
			)
		}
		if r.Cfg.ReportOptions.CSV {
			r.Report.writePercentilesEntry(res, currentTickMetrics.Metrics)
		}
		if r.Cfg.Prometheus != nil && r.Cfg.Prometheus.Enable {
			r.PromReporter.reportTick(currentTickMetrics)
		}
		r.scaleAttackers(currentTickMetrics)
		currentTickMetrics.Reported = true
		delete(r.receivedTickMetrics, res.AttackToken.Tick)
	}
}

// printErrors print uniq errors
func (r *Runner) printErrors() {
	r.L.Infof("Uniq errors:")
	for e, count := range r.uniqErrors {
		r.L.Infof("error: %s, count: %d", e, count)
	}
}

// maxRPS calculate max rps for test among ticks
func (r *Runner) maxRPS() float64 {
	f, err := os.Open(r.Report.percLogFilename)
	if err != nil {
		log.Fatal(err)
	}
	rpsSlice := make([]float64, 0)
	csvFile := csv.NewReader(f)
	for {
		line, err := csvFile.Read()
		if err == io.EOF {
			break
		}
		rps, _ := strconv.ParseFloat(line[2], 64)
		rpsSlice = append(rpsSlice, rps)
	}
	return MaxRPS(rpsSlice)
}
