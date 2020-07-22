package loaderbot

import (
	"context"
	"log"
	"os"
	"runtime"
	"sync"
	"sync/atomic"
	"time"

	"go.uber.org/ratelimit"
)

const (
	DefaultMetricsUpdateInterval = 1 * time.Second
	DefaultScheduleQueueCapacity = 10000
	DefaultResultsQueueCapacity  = 10000
)

// Controlled struct for adding test vars
type Controlled struct {
	Sleep uint64
}

// TestData shared test data
type TestData struct {
	*sync.Mutex
	Index int
	Data  interface{}
}

type nextMsg struct {
	Step uint64
}

// Runner provides test context for attacking target with constant amount of runners with a schedule
type Runner struct {
	// Name of a runner
	Name string
	// Cfg runner config
	Cfg *RunnerConfig

	attackerPrototype Attack

	targetRPS int

	currentStep   uint64
	metricsMu     *sync.Mutex
	stepMetricsMu *sync.Mutex
	stepMetrics   map[uint64]*Metrics

	currentTick   uint64
	tickMetricsMu *sync.Mutex
	tickMetrics   map[uint64]*Metrics

	rlMu *sync.Mutex
	rl   ratelimit.Limiter

	// metrics constant load metrics
	metrics *Metrics

	// TimeoutCtx test timeout ctx
	TimeoutCtx context.Context
	cancel     context.CancelFunc
	// next schedule chan to signal to attack
	next chan nextMsg

	attackersMu *sync.Mutex
	attackers   []Attack

	results    chan AttackResult
	resultsLog []AttackResult
	// uniq error messages
	uniqErrors map[string]int

	// Failed means there some errors in test
	Failed bool

	// data used to control attackers in test
	controlled Controlled

	// TestData data shared between attackers during test
	TestData *TestData

	L *Logger
}

// NewRunner creates new runner with constant amount of attackers by RunnerConfig
func NewRunner(cfg *RunnerConfig, a Attack, data *TestData) *Runner {
	r := &Runner{
		Name:              cfg.Name,
		Cfg:               cfg,
		attackerPrototype: a,
		currentStep:       0,
		targetRPS:         cfg.StartRPS,
		metricsMu:         &sync.Mutex{},
		currentTick:       0,
		stepMetricsMu:     &sync.Mutex{},
		stepMetrics:       make(map[uint64]*Metrics),
		tickMetricsMu:     &sync.Mutex{},
		tickMetrics:       make(map[uint64]*Metrics),
		metrics:           NewMetrics(),
		next:              make(chan nextMsg, DefaultScheduleQueueCapacity),
		rlMu:              &sync.Mutex{},
		rl:                ratelimit.New(cfg.StartRPS),
		attackersMu:       &sync.Mutex{},
		attackers:         make([]Attack, 0),
		results:           make(chan AttackResult, DefaultResultsQueueCapacity),
		resultsLog:        make([]AttackResult, 0),
		uniqErrors:        make(map[string]int),
		controlled:        Controlled{},
		TestData:          data,
		L:                 NewLogger(cfg).With("runner", cfg.Name),
	}
	r.validate()
	r.DefaultCfgValues()
	for i := 0; i < cfg.Attackers; i++ {
		a := r.attackerPrototype.Clone(r)
		if err := a.Setup(*r.Cfg); err != nil {
			log.Fatal(errAttackerSetup)
		}
		r.attackers = append(r.attackers, a)
	}
	r.stepMetrics[0] = NewMetrics()
	r.tickMetrics[0] = NewMetrics()
	return r
}

func (r *Runner) validate() {
	errors := r.Cfg.Validate()
	if len(errors) > 0 {
		for _, e := range errors {
			r.L.Error(e)
		}
		os.Exit(1)
	}
}

func (r *Runner) DefaultCfgValues() {
	if r.Cfg.SystemMode == OpenWorldSystem {
		// attacker will spawn goroutines for requests anyway, in this mode we are non-blocking
		r.Cfg.Attackers = 1
	}
	// constant load
	if r.Cfg.StepRPS == 0 {
		r.Cfg.StepDurationSec = 10
	}
}

// Run runs the test
func (r *Runner) Run() (float64, error) {
	r.L.Infof("waiting for %d seconds before start", r.Cfg.WaitBeforeSec)
	time.Sleep(time.Duration(r.Cfg.WaitBeforeSec) * time.Second)
	r.L.Infof("runner started, mode: %s", r.Cfg.SystemMode)
	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(r.Cfg.TestTimeSec)*time.Second)
	r.TimeoutCtx = ctx
	r.cancel = cancel
	wg := &sync.WaitGroup{}
	wg.Add(len(r.attackers))
	for atkIdx, attacker := range r.attackers {
		r.L.Infof("starting attacker: %d", atkIdx)
		switch r.Cfg.SystemMode {
		case OpenWorldSystem:
			go asyncAttack(attacker, r, wg)
		case PrivateSystem:
			go attack(attacker, r, wg)
		}
	}
	wg.Wait()
	r.handleShutdownSignal()
	r.schedule()
	r.rampUp()
	r.updateMetrics()
	r.collectResults()
	<-r.TimeoutCtx.Done()
	r.cancel()
	r.L.Infof("runner exited")
	maxRPS := r.maxRPS()
	r.L.Infof("max rps: %.2f", maxRPS)
	return maxRPS, nil
}

func (r *Runner) printErrors() {
	r.L.Infof("Uniq errors:")
	for e, count := range r.uniqErrors {
		r.L.Infof("error: %s, count: %d", e, count)
	}
}

func (r *Runner) maxRPS() float64 {
	r.metricsMu.Lock()
	defer r.metricsMu.Unlock()
	rates := make([]float64, 0)
	for _, m := range r.stepMetrics {
		rates = append(rates, m.Rate)
	}
	return MaxRPS(rates)
}

// schedule creates schedule plan for a test
func (r *Runner) schedule() {
	go func() {
		for {
			select {
			case <-r.TimeoutCtx.Done():
				r.L.Infof("schedule stopped")
				return
			default:
				r.rlMu.Lock()
				r.rl.Take()
				r.rlMu.Unlock()
				currentStep := atomic.LoadUint64(&r.currentStep)
				r.next <- nextMsg{
					Step: currentStep,
				}
			}
		}
	}()
}

// rampUp changes ratelimit options on the run, increasing by step to target rps
func (r *Runner) rampUp() {
	ticker := time.NewTicker(time.Duration(r.Cfg.StepDurationSec) * time.Second)
	go func() {
		for {
			select {
			case <-r.TimeoutCtx.Done():
				return
			case <-ticker.C:
				currentStep := atomic.LoadUint64(&r.currentStep)

				r.metricsMu.Lock()
				stepMetrics := r.stepMetrics[currentStep]
				stepMetrics.update(r)
				r.L.Infof("STEP rate [%4f -> %v], perc: 50 [%v] 95 [%v], # requests [%d], # attackers [%d], %% success [%d]",
					stepMetrics.Rate,
					r.targetRPS,
					stepMetrics.Latencies.P50,
					stepMetrics.Latencies.P95,
					stepMetrics.Requests,
					len(r.attackers),
					stepMetrics.successLogEntry(),
				)
				r.targetRPS += r.Cfg.StepRPS
				r.rlMu.Lock()
				r.rl = ratelimit.New(r.targetRPS)
				r.rlMu.Unlock()
				atomic.AddUint64(&r.currentStep, 1)
				r.stepMetrics[currentStep+1] = NewMetrics()
				r.L.Infof("next step: step -> %d, rps -> %d", currentStep+1, r.targetRPS)
				r.metricsMu.Unlock()
				r.L.Infof("current active goroutines: %d", runtime.NumGoroutine())
			}
		}
	}()
}

// collectResults collects attackers results and writes them to one of report options
func (r *Runner) collectResults() {
	go func() {
		for {
			select {
			case <-r.TimeoutCtx.Done():
				r.stepMetricsMu.Lock()
				r.printErrors()
				r.stepMetricsMu.Unlock()
				return
			case res := <-r.results:
				currentStep := atomic.LoadUint64(&r.currentStep)
				r.metricsMu.Lock()
				r.stepMetrics[currentStep].add(res)
				r.metricsMu.Unlock()

				currentTick := atomic.LoadUint64(&r.currentTick)
				r.metricsMu.Lock()
				r.tickMetrics[currentTick].add(res)
				r.metricsMu.Unlock()

				if res.doResult.Error != nil {
					r.uniqErrors[res.doResult.Error.Error()] += 1
					r.L.Debugf("attacker error: %s", res.doResult.Error)
					if r.Cfg.FailOnFirstError {
						r.Failed = true
						r.cancel()
					}
				}
				r.L.Debugf("received result: %v", res)
				r.resultsLog = append(r.resultsLog, res)
			}
		}
	}()
}

func (r *Runner) updateMetrics() {
	ticker := time.NewTicker(DefaultMetricsUpdateInterval)
	go func() {
		for {
			select {
			case <-r.TimeoutCtx.Done():
				return
			case <-ticker.C:
				r.metricsMu.Lock()
				currentTick := atomic.LoadUint64(&r.currentTick)
				tickMetics := r.tickMetrics[currentTick]
				tickMetics.update(r)
				r.L.Infof("rate [%4f -> %v], perc: 50 [%v] 95 [%v], # requests [%d], # attackers [%d], %% success [%d]",
					tickMetics.Rate,
					r.targetRPS,
					tickMetics.Latencies.P50,
					tickMetics.Latencies.P95,
					tickMetics.Requests,
					len(r.attackers),
					tickMetics.successLogEntry(),
				)
				atomic.AddUint64(&r.currentTick, 1)
				r.tickMetrics[currentTick+1] = NewMetrics()
				r.metricsMu.Unlock()
			}
		}
	}()
}
