package loaderbot

import (
	"context"
	"sync"
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
func attack(a Attack, r *Runner, wg *sync.WaitGroup) {
	wg.Done()
	for {
		select {
		case <-r.TimeoutCtx.Done():
			return
		case <-r.next:
			ctx, cancel := context.WithTimeout(r.TimeoutCtx, time.Duration(r.cfg.AttackerTimeout)*time.Second)
			defer cancel()

			tStart := time.Now()

			done := make(chan DoResult)
			go func() {
				done <- a.Do(ctx)
			}()
			var doResult DoResult
			// either get the result from the attacker or from the timeout
			select {
			case <-ctx.Done():
				doResult = DoResult{
					RequestLabel: r.name,
					Error:        errAttackDoTimedOut,
				}
			case doResult = <-done:
			}

			tEnd := time.Now()

			atkResult := AttackResult{
				begin:    tStart,
				end:      tEnd,
				elapsed:  tEnd.Sub(tStart),
				doResult: doResult,
			}
			r.results <- atkResult
		}
	}
}
