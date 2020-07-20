package loaderbot

import (
	"context"
	"log"
	"os"
	"sync"
	"sync/atomic"
	"time"

	"go.uber.org/ratelimit"
)

const (
	DefaultMetricsUpdateInterval = 1 * time.Second
	// scaling will be performed until current rate / target rate < ScaleUntilPercent
	ScaleUntilPercent             = 0.90
	DefaultScalingSkipTicks       = 1
	DefaultScalingAttackersAmount = 200
	DefaultScheduleQueueCapacity  = 100
)

type ScalingInfo struct {
	TicksInSteps map[uint64]int
}

// Controlled struct for adding test vars
type Controlled struct {
	Sleep uint64
}

type TestData struct {
	*sync.Mutex
	Index int
	Data  interface{}
}

// Runner provides test context for attacking target with constant amount of runners with a schedule
type Runner struct {
	// Name of a runner
	Name string
	// Cfg runner config
	Cfg *RunnerConfig

	attackerPrototype Attack

	stepMu      *sync.Mutex
	currentStep uint64

	targetRPS int

	stepMetricsMu *sync.Mutex
	stepMetrics   map[uint64]*Metrics

	currentTick uint64

	skipTicks     int
	tickMetricsMu *sync.Mutex
	// tickUpdateMetrics used to check precision of dynamic rps correction
	tickUpdateMetrics []*Metrics

	scalingInfo ScalingInfo

	rlMu *sync.Mutex
	rl   ratelimit.Limiter

	// metrics constant load metrics
	metrics *Metrics

	// TimeoutCtx test timeout ctx
	TimeoutCtx context.Context
	cancel     context.CancelFunc
	// next schedule chan to signal to attack
	next chan bool

	attackersMu      *sync.Mutex
	attackers        []Attack
	dynamicAttackers []Attack

	results    chan AttackResult
	resultsLog []AttackResult

	// Failed means there some errors in test
	Failed bool

	// data used to control attackers in test
	controlledMu *sync.Mutex
	controlled   Controlled

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
		stepMetricsMu:     &sync.Mutex{},
		stepMetrics:       make(map[uint64]*Metrics),
		tickMetricsMu:     &sync.Mutex{},
		currentTick:       0,
		tickUpdateMetrics: make([]*Metrics, 0),
		scalingInfo:       ScalingInfo{TicksInSteps: make(map[uint64]int)},
		metrics:           NewMetrics(),
		next:              make(chan bool, DefaultScheduleQueueCapacity),
		stepMu:            &sync.Mutex{},
		rlMu:              &sync.Mutex{},
		rl:                ratelimit.New(cfg.StartRPS),
		attackersMu:       &sync.Mutex{},
		attackers:         make([]Attack, 0),
		dynamicAttackers:  make([]Attack, 0),
		results:           make(chan AttackResult),
		resultsLog:        make([]AttackResult, 0),
		controlledMu:      &sync.Mutex{},
		controlled:        Controlled{},
		TestData:          data,
		L:                 NewLogger(cfg).With("runner", cfg.Name),
	}
	r.Validate()
	r.DefaultCfgValues()
	for i := 0; i < cfg.Attackers; i++ {
		a := r.attackerPrototype.Clone(r)
		if err := a.Setup(*r.Cfg); err != nil {
			log.Fatal(errAttackerSetup)
		}
		r.attackers = append(r.attackers, a)
	}
	// add zero tick metrics to be able to compare with previous step
	r.tickUpdateMetrics = append(r.tickUpdateMetrics, NewMetrics())
	return r
}

func (r *Runner) Validate() {
	errors := r.Cfg.Validate()
	if len(errors) > 0 {
		for _, e := range errors {
			r.L.Error(e)
		}
		os.Exit(1)
	}
}

func (r *Runner) DefaultCfgValues() {
	if r.Cfg.ScalingAttackers == 0 {
		r.Cfg.ScalingAttackers = 200
	}
	if r.Cfg.ScalingSkipTicks == 0 {
		r.Cfg.ScalingSkipTicks = 1
	}
}

// Run runs the test
func (r *Runner) Run() (float64, error) {
	r.L.Infof("waiting for %d seconds before start", r.Cfg.WaitBeforeSec)
	time.Sleep(time.Duration(r.Cfg.WaitBeforeSec) * time.Second)
	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(r.Cfg.TestTimeSec)*time.Second)
	r.TimeoutCtx = ctx
	r.cancel = cancel
	wg := &sync.WaitGroup{}
	wg.Add(len(r.attackers))
	for atkIdx, attacker := range r.attackers {
		r.L.Infof("starting attacker: %d", atkIdx)
		go attack(attacker, r, wg)
	}
	wg.Wait()
	r.schedule()
	r.rampUp()
	r.collectResults()
	r.updateMetrics()
	r.handleShutdownSignal()
	<-r.TimeoutCtx.Done()
	r.cancel()
	r.L.Infof("runner exited")
	return r.maxRPS(), nil
}

func (r *Runner) maxRPS() float64 {
	r.tickMetricsMu.Lock()
	defer r.tickMetricsMu.Unlock()
	rates := make([]float64, 0)
	for _, tickMetrics := range r.tickUpdateMetrics {
		rate := tickMetrics.Rate
		if rate > tickMetrics.TargetRate {
			rate = tickMetrics.TargetRate
		}
		rates = append(rates, rate)
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
				r.next <- true
			}
		}
	}()
}

// scaleAttackers scaling attackers to reach target rps, check last tick metrics and
// start up attackers, wait DefaultScalingSkipTicks for metrics update
func (r *Runner) scaleAttackers() {
	lastUpdateMetrics := r.tickUpdateMetrics[atomic.LoadUint64(&r.currentTick)]
	percDiff := lastUpdateMetrics.Rate / float64(r.targetRPS)
	if percDiff < ScaleUntilPercent {
		if r.skipTicks > 0 {
			r.L.Infof("scale tick skipped: %d", r.skipTicks)
			r.skipTicks--
			return
		}
		r.L.Infof("spawning dynamic attackers: %d", DefaultScalingAttackersAmount)
		wg := &sync.WaitGroup{}
		wg.Add(DefaultScalingAttackersAmount)
		for i := 0; i < DefaultScalingAttackersAmount; i++ {
			r.attackersMu.Lock()
			a := r.attackerPrototype.Clone(r)
			if err := a.Setup(*r.Cfg); err != nil {
				log.Fatal(errAttackerSetup)
			}
			r.dynamicAttackers = append(r.dynamicAttackers, a)
			r.attackersMu.Unlock()
			go attack(a, r, wg)
		}
		wg.Wait()
		r.skipTicks += DefaultScalingSkipTicks
		currentStep := atomic.LoadUint64(&r.currentStep)
		r.scalingInfo.TicksInSteps[currentStep] += 1
	}

}

// rampUp changes ratelimit options on the run, increasing by step to target rps
func (r *Runner) rampUp() {
	ticker := NewImmediateTicker(time.Duration(r.Cfg.StepDurationSec) * time.Second)
	go func() {
		for {
			select {
			case <-r.TimeoutCtx.Done():
				return
			case <-ticker.C:
				r.tickMetricsMu.Lock()
				r.targetRPS += r.Cfg.StepRPS
				r.tickMetricsMu.Unlock()
				r.rlMu.Lock()
				r.rl = ratelimit.New(r.targetRPS)
				r.rlMu.Unlock()
				currentStep := atomic.LoadUint64(&r.currentStep)
				atomic.AddUint64(&r.currentStep, 1)
				r.L.Infof("updating step: step -> %d, rps -> %d", currentStep, r.targetRPS)
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
				return
			case res := <-r.results:
				r.tickMetricsMu.Lock()
				lastTickMetrics := r.tickUpdateMetrics[len(r.tickUpdateMetrics)-1]
				lastTickMetrics.add(res)
				r.tickMetricsMu.Unlock()
				if res.doResult.Error != nil {
					r.L.Debugf("attacker error: %s", res.doResult.Error)
					r.Failed = true
					if r.Cfg.FailOnFirstError {
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
				if len(r.tickUpdateMetrics) == 0 {
					continue
				}
				r.tickMetricsMu.Lock()
				lastTickMetrics := r.tickUpdateMetrics[atomic.LoadUint64(&r.currentTick)]
				r.tickUpdateMetrics = append(r.tickUpdateMetrics, NewMetrics())
				lastTickMetrics.update(r)
				r.L.Infof("rate [%4f -> %v], mean response [%v], # requests [%d], # attackers [%d], %% success [%d]",
					lastTickMetrics.Rate,
					r.targetRPS, lastTickMetrics.meanLogEntry(),
					lastTickMetrics.Requests,
					len(r.attackers)+len(r.dynamicAttackers),
					lastTickMetrics.successLogEntry(),
				)

				if r.Cfg.DynamicAttackers {
					r.scaleAttackers()
				}
				atomic.StoreUint64(&r.currentTick, r.currentTick+1)
				r.tickMetricsMu.Unlock()
			}
		}
	}()
}
