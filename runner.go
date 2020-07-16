package loaderbot

import (
	"context"
	"go.uber.org/ratelimit"
	"sync"
	"time"
)

const (
	DefaultMetricsUpdateInterval = 1 * time.Second
)

// Runner provides test context for attacking target with constant amount of runners with a schedule
type Runner struct {
	name string
	cfg  *RunnerConfig

	stepMu      *sync.Mutex
	currentStep int
	currentRPS  int
	stepMetrics map[int]*Metrics
	rl          ratelimit.Limiter

	metrics *Metrics

	TimeoutCtx context.Context
	cancel     context.CancelFunc
	next       chan bool

	attackers []Attack

	results    chan AttackResult
	resultsLog []AttackResult

	L *Logger
}

// NewRunner creates new runner with constant amount of attackers by RunnerConfig
func NewRunner(cfg *RunnerConfig) *Runner {
	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(cfg.Timeout)*time.Second)
	r := &Runner{
		name:        cfg.Name,
		cfg:         cfg,
		currentStep: 0,
		currentRPS:  cfg.StartRPS,
		stepMetrics: make(map[int]*Metrics, 0),
		metrics:     NewMetrics(),
		TimeoutCtx:  ctx,
		cancel:      cancel,
		next:        make(chan bool, 100),
		stepMu:      &sync.Mutex{},
		rl:          ratelimit.New(cfg.StartRPS),
		attackers:   make([]Attack, 0),
		results:     make(chan AttackResult),
		resultsLog:  make([]AttackResult, 0),
		L:           NewLogger().With("runner", cfg.Name),
	}
	for i := 0; i < cfg.Attackers; i++ {
		r.attackers = append(r.attackers, NewAttacker(i, r))
	}
	return r
}

// Run runs the test
func (r *Runner) Run() error {
	for atkIdx, attacker := range r.attackers {
		r.L.Infof("starting attacker: %d", atkIdx)
		go attack(attacker, r, atkIdx)
	}
	r.L.Infof("waiting for %d seconds before start", r.cfg.WaitBefore)
	time.Sleep(time.Duration(r.cfg.WaitBefore) * time.Second)
	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(r.cfg.Timeout)*time.Second)
	r.TimeoutCtx = ctx
	r.cancel = cancel
	r.schedule()
	r.rampUp()
	r.collectResults()
	r.updateMetrics()
	r.handleShutdownSignal()
	<-r.TimeoutCtx.Done()
	return nil
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
				r.rl.Take()
				r.next <- true
			}
		}
	}()
}

// rampUp changes ratelimit options on the run, increasing by step to target rps
func (r *Runner) rampUp() {
	go func() {
		for {
			select {
			case <-r.TimeoutCtx.Done():
				return
			case <-time.Tick(time.Duration(r.cfg.StepDuration) * time.Second):
				r.stepMu.Lock()
				r.currentStep += 1
				r.currentRPS += r.cfg.StepRPS
				r.rl = ratelimit.New(r.currentRPS)
				r.stepMetrics[r.currentStep] = NewMetrics()
				r.stepMetrics[r.currentStep].update()
				r.L.Infof("updating step: step->%d, rps->%d", r.currentStep, r.currentRPS)
				r.stepMu.Unlock()
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
				r.L.Debugf("received result: %v", res)
				r.resultsLog = append(r.resultsLog, res)
			}
		}
	}()
}

func (r *Runner) updateMetrics() {
	go func() {
		for {
			select {
			case <-r.TimeoutCtx.Done():
				return
			case <-time.Tick(DefaultMetricsUpdateInterval):
				if _, ok := r.stepMetrics[r.currentStep]; ok {
					r.stepMetrics[r.currentStep].update()
					r.L.Infof("rate [%4f -> %v], mean response [%v], # requests [%d], # attackers [%d], %% success [%d]",
						r.stepMetrics[r.currentStep].Rate,
						r.currentRPS, r.stepMetrics[r.currentStep].meanLogEntry(),
						r.stepMetrics[r.currentStep].Requests,
						len(r.attackers),
						r.stepMetrics[r.currentStep].successLogEntry(),
					)
				}
			}
		}
	}()
}
