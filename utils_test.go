/*
 * // Copyright 2020 Insolar Network Ltd.
 * // All rights reserved.
 * // This material is licensed under the Insolar License version 1.0,
 * // available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.
 */

package loaderbot

import (
	"sync/atomic"
	"time"
)

func serviceErrorAfter(se chan bool, t time.Duration) {
	go func() {
		time.Sleep(t)
		se <- true
	}()
}

type ControllableConfig struct {
	R               *Runner
	ControlChan     chan bool
	AttackersAmount int
}

func withControllableAttackers(cfg ControllableConfig) {
	attackers := make([]Attack, 0)
	for i := 0; i < cfg.AttackersAmount; i++ {
		attackers = append(attackers, NewControlMockAttacker(i, cfg.ControlChan, cfg.R))
	}
	cfg.R.attackers = attackers
}

// nolint
type ServiceLatencyChangeConfig struct {
	R             *Runner
	Interval      time.Duration
	LatencyStepMs int64
	Times         int
	LatencyFlag   int
}

const (
	increaseLatency = iota
	decreaseLatency
)

// nolint
func changeAttackersLatency(cfg ServiceLatencyChangeConfig) {
	for i := 0; i < cfg.Times; i++ {
		if cfg.LatencyFlag == increaseLatency {
			atomic.AddInt64(&cfg.R.controlled.Sleep, cfg.LatencyStepMs)
		}
		if cfg.LatencyFlag == decreaseLatency {
			atomic.AddInt64(&cfg.R.controlled.Sleep, -cfg.LatencyStepMs)
		}
		time.Sleep(cfg.Interval)
	}
	cfg.R.L.Infof("=== done changing latency ===")
	cfg.R.L.Infof("=== keeping latency constant for new attackers ===")
}
