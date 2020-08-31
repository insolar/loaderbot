/*
 * // Copyright 2020 Insolar Network Ltd.
 * // All rights reserved.
 * // This material is licensed under the Insolar License version 1.0,
 * // available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.
 */

package loaderbot

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"runtime"
	"sync"
	"sync/atomic"
	"time"

	"github.com/prometheus/client_golang/prometheus/promhttp"
	"go.uber.org/ratelimit"
)

const (
	DefaultResultsQueueCapacity = 100_000
	MetricsLogFile              = "requests_%s_%s_%d.csv"
	PercsLogFile                = "percs_%s_%s_%d.csv"
	ReportGraphFile             = "percs_%s_%s_%d.html"
)

var (
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
	L              *Logger
}

// NewRunner creates new runner with constant amount of attackers by RunnerConfig
func NewRunner(cfg *RunnerConfig, a Attack, data interface{}) *Runner {
	cfg.Validate()
	cfg.DefaultCfgValues()
	r := &Runner{
		Name:                  cfg.Name,
		Cfg:                   cfg,
		attackerPrototype:     a,
		targetRPS:             cfg.StartRPS,
		next:                  make(chan attackToken),
		rl:                    ratelimit.New(cfg.StartRPS),
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
		http.Handle("/metrics", promhttp.Handler())
		var port int
		if cfg.Prometheus.Port == 0 {
			port = 2112
		}
		// nolint
		go http.ListenAndServe(fmt.Sprintf(":%d", port), nil)
	}
	return r
}

// Run runs the test
func (r *Runner) Run(serverCtx context.Context) (float64, error) {
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
		switch r.Cfg.SystemMode {
		case OpenWorldSystem:
			go asyncAttack(attacker, r)
		case PrivateSystem:
			r.L.Debugf("starting attacker: %d", atkIdx)
			go attack(attacker, r)
		}
	}
	r.handleShutdownSignal()
	r.schedule()
	r.collectResults()
	<-r.TimeoutCtx.Done()
	r.CancelFunc()
	r.L.Infof("runner exited")
	maxRPS := r.maxRPS()
	r.L.Infof("max rps: %.2f", maxRPS)
	if r.Cfg.ReportOptions.CSV {
		r.Report.flushLogs()
		r.Report.plot()
	}
	return maxRPS, nil
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
		for {
			select {
			case <-r.TimeoutCtx.Done():
				r.L.Infof("total requests fired: %d", totalRequestsFired)
				close(r.next)
				return
			default:
				r.rl.Take()
				r.next <- attackToken{
					TargetRPS: r.targetRPS,
					Step:      currentStep,
					Tick:      currentTick,
				}
				totalRequestsFired++
				requestsFiredInTick++
				if requestsFiredInTick == r.targetRPS {
					currentTick += 1
					requestsFiredInTick = 0
					r.L.Infof("current active goroutines: %d", runtime.NumGoroutine())
					if currentTick%ticksInStep == 0 {
						r.targetRPS += r.Cfg.StepRPS
						r.rl = ratelimit.New(r.targetRPS)
						currentStep += 1
						r.L.Infof("next step: step -> %d, rps -> %d", currentStep, r.targetRPS)
					}
				}
			}
		}
	}()
}

// collectResults collects attackers Results and writes them to one of report options
func (r *Runner) collectResults() {
	go func() {
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
		fmt.Printf("success: [ %.2f < %.2f ]", currentTickMetrics.Metrics.Success, r.Cfg.SuccessRatio)
		if currentTickMetrics.Metrics.Success < r.Cfg.SuccessRatio {
			atomic.AddInt64(&r.Failed, 1)
			r.CancelFunc()
		}
		r.L.Infof(
			"step: %d, tick: %d, rate [%.4f -> %v], perc: 50 [%v] 95 [%v] 99 [%v], # requests [%d], %% success [%d]",
			res.AttackToken.Step,
			res.AttackToken.Tick,
			currentTickMetrics.Metrics.Rate,
			res.AttackToken.TargetRPS,
			currentTickMetrics.Metrics.Latencies.P50,
			currentTickMetrics.Metrics.Latencies.P95,
			currentTickMetrics.Metrics.Latencies.P99,
			currentTickMetrics.Metrics.Requests,
			currentTickMetrics.Metrics.successLogEntry(),
		)
		if r.Cfg.ReportOptions.CSV {
			r.Report.writePercentilesEntry(res, currentTickMetrics.Metrics)
		}
		if r.Cfg.Prometheus != nil && r.Cfg.Prometheus.Enable {
			r.PromReporter.reportTick(currentTickMetrics)
		}
		currentTickMetrics.Reported = true
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
	r.receivedTickMetricsMu.Lock()
	defer r.receivedTickMetricsMu.Unlock()
	rates := make([]float64, 0)
	for _, m := range r.receivedTickMetrics {
		rates = append(rates, m.Metrics.Rate)
	}
	return MaxRPS(rates)
}
