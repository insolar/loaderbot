/*
 * // Copyright 2020 Insolar Network Ltd.
 * // All rights reserved.
 * // This material is licensed under the Insolar License version 1.0,
 * // available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.
 */

package loaderbot

import (
	"context"
	"testing"
	"time"
)

func DefaultRunnerCfg() *RunnerConfig {
	return &RunnerConfig{
		Name:            "test_runner",
		Attackers:       1,
		AttackerTimeout: 1,
		StartRPS:        20,
		StepDurationSec: 2,
		StepRPS:         1,
		TestTimeSec:     5,
		ReportOptions: &ReportOptions{
			CSV: false,
			PNG: false,
		},
	}
}

func TestCommonAttackSuccess(t *testing.T) {
	r := NewRunner(DefaultRunnerCfg(), &ControlAttackerMock{}, nil)
	r.controlled.Sleep = 10
	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(r.Cfg.TestTimeSec)*time.Second)
	r.TimeoutCtx = ctx
	r.CancelFunc = cancel

	// sync
	go attack(r.attackers[0], r)
	r.next <- attackToken{
		Step: 1,
		Tick: 1,
	}
	res := <-r.results
	if got, want := res.DoResult.Error, ""; got != want {
		t.Fatalf("got %v want %v", got, want)
	}
	if got, want := int(res.Elapsed), int(r.controlled.Sleep); got < want {
		t.Fatalf("got %v want >= %v", got, want)
	}
}

func TestCommonAttackTimeout(t *testing.T) {
	r := NewRunner(DefaultRunnerCfg(), &ControlAttackerMock{}, nil)
	r.controlled.Sleep = 2000
	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(r.Cfg.TestTimeSec)*time.Second)
	r.TimeoutCtx = ctx
	r.CancelFunc = cancel

	// sync
	go attack(r.attackers[0], r)
	r.next <- attackToken{
		Step: 1,
		Tick: 1,
	}
	res := <-r.results
	if got, want := res.DoResult.Error, errAttackDoTimedOut; got != want {
		t.Fatalf("got %v want %v", got, want)
	}
}
