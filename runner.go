/*
 * // Copyright 2020 Insolar Network Ltd.
 * // All rights reserved.
 * // This material is licensed under the Insolar License version 1.0,
 * // available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.
 */

package loaderbot

import (
	"context"
	"encoding/csv"
	"fmt"
	"log"
	"runtime"
	"strconv"
	"sync"
	"sync/atomic"
	"time"

	"github.com/google/uuid"
	"go.uber.org/ratelimit"
)

const (
	DefaultScheduleQueueCapacity = 100000
	DefaultResultsQueueCapacity  = 100000
	MetricsLogFile               = "requests_%s_%s_%d.csv"
	PercsLogFile                 = "percs_%s_%s_%d.csv"
	ReportGraphFile              = "percs_%s_%s_%d.html"
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
	Name  string
	runId string
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
	// raw Results log
	rawResultsLog      []AttackResult
	metricsLogFilename string
	// raw attack Results log file
	metricsLogFile      *csv.Writer
	percsReportFilename string
	percLogFilename     string
	// aggregated per second P50/95/99 percentiles of response time log
	percLogFile *csv.Writer
	// uniq error messages
	uniqErrors map[string]int
	// Failed means there some errors in test
	Failed int64
	// data used to control attackers in test
	controlled Controlled
	// TestData data shared between attackers during test
	TestData interface{}
	L        *Logger
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
		next:                  make(chan attackToken, DefaultScheduleQueueCapacity),
		rl:                    ratelimit.New(cfg.StartRPS),
		attackers:             make([]Attack, 0),
		results:               make(chan AttackResult, DefaultResultsQueueCapacity),
		OutResults:            make(chan []AttackResult, DefaultResultsQueueCapacity),
		rawResultsLog:         make([]AttackResult, 0),
		receivedTickMetricsMu: &sync.Mutex{},
		receivedTickMetrics:   make(map[int]*TickMetrics),
		uniqErrors:            make(map[string]int),
		controlled:            Controlled{},
		TestData:              data,
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
		r.runId = uuid.New().String()
		tn := time.Now().Unix()
		r.metricsLogFilename = fmt.Sprintf(MetricsLogFile, cfg.Name, r.runId, tn)
		r.percLogFilename = fmt.Sprintf(PercsLogFile, cfg.Name, r.runId, tn)
		r.percsReportFilename = fmt.Sprintf(ReportGraphFile, r.Name, r.runId, tn)
		r.metricsLogFile = csv.NewWriter(CreateFileOrReplace(r.metricsLogFilename))
		r.percLogFile = csv.NewWriter(CreateFileOrReplace(r.percLogFilename))
		_ = r.metricsLogFile.Write(ResultsCsvHeader)
		_ = r.percLogFile.Write(PercsCsvHeader)
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
	wg := &sync.WaitGroup{}
	wg.Add(len(r.attackers))
	for atkIdx, attacker := range r.attackers {
		switch r.Cfg.SystemMode {
		case OpenWorldSystem:
			go asyncAttack(attacker, r, wg)
		case PrivateSystem:
			r.L.Debugf("starting attacker: %d", atkIdx)
			go attack(attacker, r, wg)
		}
	}
	r.handleShutdownSignal()
	r.schedule()
	r.collectResults()
	<-r.TimeoutCtx.Done()
	r.CancelFunc()
	wg.Wait()
	r.L.Infof("runner exited")
	maxRPS := r.maxRPS()
	r.L.Infof("max rps: %.2f", maxRPS)
	if r.Cfg.ReportOptions.CSV {
		r.flushLogs()
	}
	r.report()
	return maxRPS, nil
}

func (r *Runner) report() {
	if r.Cfg.ReportOptions.PNG {
		r.L.Infof("reporting graphs: %s", r.percLogFilename)
		chart, err := PercsChart(r.percLogFilename, r.Name)
		if err != nil {
			r.L.Error(err)
			return
		}
		RenderEChart(chart, r.percsReportFilename)
		// html2png(r.percsReportFilename)
	}
}

func (r *Runner) flushLogs() {
	r.percLogFile.Flush()
	r.metricsLogFile.Flush()
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
					if r.Cfg.FailOnFirstError {
						atomic.AddInt64(&r.Failed, 1)
					}
					errorForReport = res.DoResult.Error
				}

				if r.Cfg.ReportOptions.CSV {
					r.writeResultEntry(res, errorForReport)
				}

				r.rawResultsLog = append(r.rawResultsLog, res)
				r.processTickMetrics(res)
			}
		}
	}()
}

// processTickMetrics add attack result to tick metrics, if it's last result in tick then report
func (r *Runner) processTickMetrics(res AttackResult) {
	// if no such tick, create new TickMetrics
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
		r.receivedTickMetricsMu.Lock()
		for _, s := range currentTickMetrics.Samples {
			if s.DoResult.Error != "" && r.Cfg.FailOnFirstError {
				r.CancelFunc()
			}
			currentTickMetrics.Metrics.add(s)
		}
		currentTickMetrics.Metrics.update()
		r.receivedTickMetricsMu.Unlock()
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
			r.writePercentilesEntry(res, currentTickMetrics.Metrics)
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

func (r *Runner) writeResultEntry(res AttackResult, errorMsg string) {
	_ = r.metricsLogFile.Write([]string{
		res.DoResult.RequestLabel,
		strconv.Itoa(int(res.Begin.UnixNano())),
		strconv.Itoa(int(res.End.UnixNano())),
		res.Elapsed.String(),
		string(res.DoResult.StatusCode),
		errorMsg,
	})
}

func (r *Runner) writePercentilesEntry(res AttackResult, tickMetrics *Metrics) {
	_ = r.percLogFile.Write([]string{
		res.DoResult.RequestLabel,
		strconv.Itoa(res.AttackToken.Tick),
		strconv.Itoa(int(tickMetrics.Rate)),
		strconv.Itoa(int(tickMetrics.Latencies.P50.Milliseconds())),
		strconv.Itoa(int(tickMetrics.Latencies.P95.Milliseconds())),
		strconv.Itoa(int(tickMetrics.Latencies.P99.Milliseconds())),
	})
}
