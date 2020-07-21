package loaderbot

import (
	"context"
	"sync"
	"testing"
	"time"
)

func DefaultRunnerCfg() *RunnerConfig {
	return &RunnerConfig{
		Name:            "abc",
		Attackers:       1,
		AttackerTimeout: 1,
		StartRPS:        20,
		StepDurationSec: 5,
		StepRPS:         5,
		TestTimeSec:     60,
		WaitBeforeSec:   0,
	}
}

func TestAttackSuccess(t *testing.T) {
	r := NewRunner(DefaultRunnerCfg(), &ControlAttackerMock{}, nil)
	r.controlled.Sleep = 10
	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(r.Cfg.TestTimeSec)*time.Second)
	r.TimeoutCtx = ctx
	r.cancel = cancel

	// sync
	wg := &sync.WaitGroup{}
	wg.Add(1)
	go attack(r.attackers[0], r, wg)
	wg.Wait()

	r.next <- nextMsg{
		Step: 1,
	}
	res := <-r.results
	if got, want := res.doResult.Error, error(nil); got != want {
		t.Fatalf("got %v want %v", got, want)
	}
	if got, want := int(res.elapsed), int(r.controlled.Sleep); got < want {
		t.Fatalf("got %v want >= %v", got, want)
	}
	// async
	wg2 := &sync.WaitGroup{}
	wg2.Add(1)
	go asyncAttack(r.attackers[0], r, wg2)
	wg2.Wait()

	r.next <- nextMsg{
		Step: 1,
	}
	res2 := <-r.results
	if got, want := res2.doResult.Error, error(nil); got != want {
		t.Fatalf("got %v want %v", got, want)
	}
	if got, want := int(res.elapsed), int(r.controlled.Sleep); got < want {
		t.Fatalf("got %v want >= %v", got, want)
	}
}

func TestAttackTimeout(t *testing.T) {
	r := NewRunner(DefaultRunnerCfg(), &ControlAttackerMock{}, nil)
	r.controlled.Sleep = 2000
	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(r.Cfg.TestTimeSec)*time.Second)
	r.TimeoutCtx = ctx
	r.cancel = cancel

	// sync
	wg := &sync.WaitGroup{}
	wg.Add(1)
	go attack(r.attackers[0], r, wg)
	wg.Wait()

	r.next <- nextMsg{
		Step: 1,
	}
	res := <-r.results
	if got, want := res.doResult.Error, errAttackDoTimedOut; got != want {
		t.Fatalf("got %v want %v", got, want)
	}

	// async
	wg2 := &sync.WaitGroup{}
	wg2.Add(1)
	go attack(r.attackers[0], r, wg2)
	wg2.Wait()

	r.next <- nextMsg{
		Step: 1,
	}
	res2 := <-r.results
	if got, want := res2.doResult.Error, errAttackDoTimedOut; got != want {
		t.Fatalf("got %v want %v", got, want)
	}
}
