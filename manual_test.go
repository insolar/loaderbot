/*
 * // Copyright 2020 Insolar Network Ltd.
 * // All rights reserved.
 * // This material is licensed under the Insolar License version 1.0,
 * // available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.
 */

package loaderbot

import (
	"context"
	"sync/atomic"
	"testing"
	"time"
)

func TestDynamicLatencyAsync(t *testing.T) {
	t.Skip("only manual run")
	r := NewRunner(&RunnerConfig{
		Name:            "test_runner",
		SystemMode:      OpenWorldSystem,
		AttackerTimeout: 25,
		StartRPS:        100,
		StepDurationSec: 5,
		StepRPS:         200,
		TestTimeSec:     60,
		ReportOptions: &ReportOptions{
			CSV: true,
			PNG: true,
		},
	}, &ControlAttackerMock{}, nil)
	r.controlled.Sleep = 10000

	latCfg := ServiceLatencyChangeConfig{
		R:             r,
		Interval:      5 * time.Second,
		LatencyStepMs: 1000,
		Times:         30,
		LatencyFlag:   decreaseLatency,
	}
	go changeAttackersLatency(latCfg)
	_, _ = r.Run(context.TODO())
}

func TestDynamicLatencySync(t *testing.T) {
	t.Skip("only manual run")
	r := NewRunner(&RunnerConfig{
		Name:            "test_runner",
		SystemMode:      PrivateSystem,
		Attackers:       5000,
		AttackerTimeout: 25,
		StartRPS:        10,
		StepDurationSec: 3,
		StepRPS:         20,
		TestTimeSec:     60,
		ReportOptions: &ReportOptions{
			CSV: true,
			PNG: true,
		},
	}, &ControlAttackerMock{}, nil)
	r.controlled.Sleep = 20000

	latCfg := ServiceLatencyChangeConfig{
		R:             r,
		Interval:      5 * time.Second,
		LatencyStepMs: 1800,
		Times:         30,
		LatencyFlag:   decreaseLatency,
	}
	go changeAttackersLatency(latCfg)
	_, _ = r.Run(context.TODO())
}

func TestRunnerRealServiceAttack(t *testing.T) {
	t.Skip("only manual run")
	r := NewRunner(&RunnerConfig{
		TargetUrl:       "https://clients5.google.com/pagead/drt/dn/",
		Name:            "test_runner",
		SystemMode:      PrivateSystem,
		Attackers:       3000,
		AttackerTimeout: 5,
		StartRPS:        1000,
		StepDurationSec: 5,
		StepRPS:         3000,
		TestTimeSec:     60,
	}, &HTTPAttackerExample{}, nil)
	_, _ = r.Run(context.TODO())
}

func TestAllJitter(t *testing.T) {
	t.Skip("only manual run")
	r := NewRunner(&RunnerConfig{
		Name:            "test_runner_open_world_decrease",
		SystemMode:      OpenWorldSystem,
		AttackerTimeout: 25,
		StartRPS:        100,
		StepDurationSec: 5,
		StepRPS:         200,
		TestTimeSec:     60,
		ReportOptions: &ReportOptions{
			CSV: true,
			PNG: true,
		},
	}, &ControlAttackerMock{}, nil)
	atomic.AddInt64(&r.controlled.Sleep, 10000)

	latCfg := ServiceLatencyChangeConfig{
		R:             r,
		Interval:      5 * time.Second,
		LatencyStepMs: 500,
		Times:         12,
		LatencyFlag:   decreaseLatency,
	}
	go changeAttackersLatency(latCfg)
	_, _ = r.Run(context.TODO())

	r2 := NewRunner(&RunnerConfig{
		Name:            "test_runner_open_world_jitter",
		SystemMode:      OpenWorldSystem,
		AttackerTimeout: 25,
		StartRPS:        100,
		StepDurationSec: 5,
		StepRPS:         200,
		TestTimeSec:     60,
		ReportOptions: &ReportOptions{
			CSV: true,
			PNG: true,
		},
	}, &ControlAttackerMock{}, nil)
	atomic.AddInt64(&r2.controlled.Sleep, 10000)

	go func() {
		for i := 0; i < 300; i++ {
			time.Sleep(100 * time.Millisecond)
			atomic.AddInt64(&r2.controlled.Sleep, -9900)
			time.Sleep(100 * time.Millisecond)
			atomic.AddInt64(&r2.controlled.Sleep, 9900)
		}
	}()
	_, _ = r2.Run(context.TODO())

	r3 := NewRunner(&RunnerConfig{
		Name:            "test_runner_private_decrease",
		SystemMode:      PrivateSystem,
		Attackers:       5000,
		AttackerTimeout: 25,
		StartRPS:        10,
		StepDurationSec: 3,
		StepRPS:         20,
		TestTimeSec:     60,
		ReportOptions: &ReportOptions{
			CSV: true,
			PNG: true,
		},
	}, &ControlAttackerMock{}, nil)
	atomic.AddInt64(&r3.controlled.Sleep, 20000)

	latCfg3 := ServiceLatencyChangeConfig{
		R:             r3,
		Interval:      5 * time.Second,
		LatencyStepMs: 1800,
		Times:         12,
		LatencyFlag:   decreaseLatency,
	}
	go changeAttackersLatency(latCfg3)
	_, _ = r3.Run(context.TODO())

	r4 := NewRunner(&RunnerConfig{
		Name:            "test_runner_private_jitter",
		SystemMode:      PrivateSystem,
		Attackers:       5000,
		AttackerTimeout: 25,
		StartRPS:        10,
		StepDurationSec: 3,
		StepRPS:         20,
		TestTimeSec:     60,
		ReportOptions: &ReportOptions{
			CSV: true,
			PNG: true,
		},
	}, &ControlAttackerMock{}, nil)
	atomic.AddInt64(&r4.controlled.Sleep, 10000)

	go func() {
		for i := 0; i < 300; i++ {
			time.Sleep(100 * time.Millisecond)
			atomic.AddInt64(&r4.controlled.Sleep, -19900)
			time.Sleep(100 * time.Millisecond)
			atomic.AddInt64(&r4.controlled.Sleep, 19900)
		}
	}()
	_, _ = r4.Run(context.TODO())
}

func TestLeak(t *testing.T) {
	t.Skip("only manual run")
	for i := 0; i < 10; i++ {
		r := NewRunner(&RunnerConfig{
			Name:            "test_runner_open_world_decrease",
			SystemMode:      PrivateSystem,
			Attackers:       300,
			AttackerTimeout: 25,
			StartRPS:        10,
			StepDurationSec: 5,
			StepRPS:         30,
			TestTimeSec:     120,
			ReportOptions: &ReportOptions{
				CSV: true,
				PNG: true,
			},
		}, &ControlAttackerMock{}, nil)
		atomic.AddInt64(&r.controlled.Sleep, 500)
		_, _ = r.Run(context.Background())
	}
}

func TestRunnerNginxStaticAttackFastHTTP(t *testing.T) {
	t.Skip("only manual run")
	go pprofTrace("fast_http", 40)
	// go tool trace -http=':8081' ${FILENAME}
	r := NewRunner(&RunnerConfig{
		TargetUrl:        "http://52.186.11.217:8080/static.html",
		Name:             "nginx_test",
		SystemMode:       PrivateSystem,
		Attackers:        2000,
		AttackerTimeout:  10,
		StartRPS:         1000,
		StepDurationSec:  5,
		StepRPS:          2000,
		TestTimeSec:      40,
		FailOnFirstError: true,
	}, &FastHTTPAttackerExample{}, nil)
	_, _ = r.Run(context.TODO())
}

func TestRunnerNginxStaticAttackDefaultHTTP(t *testing.T) {
	t.Skip("only manual run")
	go pprofTrace("default_http", 40)
	// go tool trace -http=':8081' ${FILENAME}
	r := NewRunner(&RunnerConfig{
		TargetUrl:       "http://52.186.11.217:8080/static.html",
		Name:            "nginx_test",
		SystemMode:      PrivateSystem,
		Attackers:       3000,
		AttackerTimeout: 5,
		StartRPS:        1000,
		StepDurationSec: 5,
		StepRPS:         200,
		TestTimeSec:     40,
	}, &HTTPAttackerExample{}, nil)
	_, _ = r.Run(context.TODO())
}
