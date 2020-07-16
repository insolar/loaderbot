package loaderbot

import (
	"context"
	"time"
)

// Attack must be implemented by a service client.
type Attack interface {
	// Setup should establish the connection to the service
	// It may want to access the Config of the Runner.
	Setup(c RunnerConfig) error
	// Do performs one request and is executed in a separate goroutine.
	// The context is used to cancel the request on timeout.
	Do(ctx context.Context) DoResult
	// Teardown can be used to close the connection to the service
	Teardown() error
	// Clone should return a fresh new Attack
	// Make sure the new Attack has values for shared struct fields initialized at Setup.
	Clone(r *Runner) Attack
}

// attack receives schedule signal and attacks target calling Do() method, returning AttackResult with timings
func attack(a Attack, r *Runner, num int) {
	l := r.L.Clone()
	ll := *l.With("attacker", num)
	for {
		select {
		case <-r.TimeoutCtx.Done():
			ll.Infof("stopping attacker")
			return
		case <-r.next:
			ctx, _ := context.WithTimeout(r.TimeoutCtx, time.Duration(r.cfg.AttackTimeout)*time.Second)

			ll.Debug("attacking")

			tStart := time.Now()
			doResult := a.Do(ctx)
			tEnd := time.Now()
			elapsed := tStart.Sub(tStart)

			atkResult := AttackResult{
				begin:    tStart,
				end:      tEnd,
				elapsed:  elapsed,
				doResult: doResult,
			}
			if _, ok := r.stepMetrics[r.currentStep]; ok {
				r.stepMetrics[r.currentStep].add(atkResult)
			}
			r.results <- atkResult
		}
	}
}
