/*
 * // Copyright 2020 Insolar Network Ltd.
 * // All rights reserved.
 * // This material is licensed under the Insolar License version 1.0,
 * // available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.
 */

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
	// The context is used to CancelFunc the request on timeout.
	Do(ctx context.Context) DoResult
	// Teardown can be used to close the connection to the service
	Teardown() error
	// Clone should return a fresh new Attack
	// Make sure the new Attack has values for shared struct fields initialized at Setup.
	Clone(r *Runner) Attack
}

// attack receives schedule signal and attacks target calling Do() method, returning AttackResult with timings
func attack(a Attack, r *Runner) {
	for nextMsg := range r.next {
		token := nextMsg
		requestCtx, requestCtxCancel := context.WithTimeout(context.Background(), time.Duration(r.Cfg.AttackerTimeout)*time.Second)

		tStart := time.Now()

		done := make(chan DoResult, 1)
		var doResult DoResult
		go func() {
			select {
			case <-r.TimeoutCtx.Done():
				requestCtxCancel()
				return
			case <-requestCtx.Done():
			case done <- a.Do(requestCtx):
			}
		}()
		// either get the result from the attacker or from the timeout
		select {
		case <-r.TimeoutCtx.Done():
			requestCtxCancel()
			return
		case <-requestCtx.Done():
			doResult = DoResult{
				RequestLabel: r.Name,
				Error:        errAttackDoTimedOut,
			}
		case doResult = <-done:
		}

		tEnd := time.Now()

		atkResult := AttackResult{
			AttackToken: token,
			Begin:       tStart,
			End:         tEnd,
			Elapsed:     tEnd.Sub(tStart),
			DoResult:    doResult,
		}
		requestCtxCancel()
		if err := a.Teardown(); err != nil {
			r.L.Infof("teardown failed: %s", err)
		}
		r.results <- atkResult
	}
}
