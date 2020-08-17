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

	"github.com/stretchr/testify/require"
)

func TestPrivateSystemRunnerSuccess(t *testing.T) {
	r := NewRunner(&RunnerConfig{
		Name:            "test_runner",
		Attackers:       10,
		AttackerTimeout: 1,
		StartRPS:        8,
		StepDurationSec: 5,
		StepRPS:         2,
		TestTimeSec:     10,
		ReportOptions: &ReportOptions{
			CSV: false,
			PNG: false,
		},
	}, &ControlAttackerMock{}, nil)
	_, err := r.Run(context.TODO())
	require.NoError(t, err)
}

func TestOpenWorldSystemRunnerSuccess(t *testing.T) {
	r := NewRunner(&RunnerConfig{
		Name:            "test_runner",
		SystemMode:      OpenWorldSystem,
		Attackers:       300,
		AttackerTimeout: 1,
		StartRPS:        100,
		StepDurationSec: 10,
		StepRPS:         20,
		TestTimeSec:     5,
		ReportOptions: &ReportOptions{
			CSV: false,
			PNG: false,
		},
	}, &ControlAttackerMock{}, nil)
	_, err := r.Run(context.TODO())
	require.NoError(t, err)
}

func TestMultipleRunnersSuccess(t *testing.T) {
	cfg := &RunnerConfig{
		Name:            "test_runner",
		Attackers:       1,
		AttackerTimeout: 1,
		StartRPS:        1,
		StepDurationSec: 5,
		StepRPS:         2,
		TestTimeSec:     1,
		ReportOptions: &ReportOptions{
			CSV: false,
			PNG: false,
		},
	}
	r := NewRunner(cfg, &ControlAttackerMock{}, nil)
	_, err := r.Run(context.TODO())
	require.NoError(t, err)
	r2 := NewRunner(cfg, &ControlAttackerMock{}, nil)
	_, err2 := r2.Run(context.TODO())
	require.NoError(t, err2)
	cfg.SystemMode = OpenWorldSystem
	r3 := NewRunner(cfg, &ControlAttackerMock{}, nil)
	_, err3 := r3.Run(context.TODO())
	require.NoError(t, err3)
	r4 := NewRunner(cfg, &ControlAttackerMock{}, nil)
	_, err4 := r4.Run(context.TODO())
	require.NoError(t, err4)
}

func TestRunnerFailOnFirstError(t *testing.T) {
	r := NewRunner(&RunnerConfig{
		Name:             "test_runner",
		Attackers:        10,
		AttackerTimeout:  1,
		StartRPS:         1,
		StepDurationSec:  5,
		StepRPS:          2,
		TestTimeSec:      5,
		FailOnFirstError: true,
		ReportOptions: &ReportOptions{
			CSV: false,
			PNG: false,
		},
	}, &ControlAttackerMock{}, nil)
	serviceError := make(chan bool)
	cfg := ControllableConfig{
		R:               r,
		ControlChan:     serviceError,
		AttackersAmount: 3,
	}
	withControllableAttackers(cfg)
	serviceErrorAfter(serviceError, 1*time.Nanosecond)
	_, _ = r.Run(context.TODO())
	require.Equal(t, int64(1), r.Failed)
}

func TestRunnerHangedRequestsAfterTimeoutNoError(t *testing.T) {
	r := NewRunner(&RunnerConfig{
		Name:             "test_runner",
		SystemMode:       PrivateSystem,
		Attackers:        1,
		AttackerTimeout:  5,
		StartRPS:         1,
		StepDurationSec:  5,
		StepRPS:          2,
		TestTimeSec:      2,
		FailOnFirstError: true,
		ReportOptions: &ReportOptions{
			CSV: false,
			PNG: false,
		},
	}, &ControlAttackerMock{}, nil)
	// request still hangs when the test ends, but it's not an error because test has ended
	r.controlled.Sleep = 5000
	_, err := r.Run(context.TODO())
	require.NoError(t, err)
	require.Empty(t, r.uniqErrors)
}

func TestPrivateSystemRunnerIsSync(t *testing.T) {
	rps := 100
	r := NewRunner(&RunnerConfig{
		Name:            "test_runner",
		SystemMode:      PrivateSystem,
		Attackers:       100,
		AttackerTimeout: 5,
		StartRPS:        rps,
		StepDurationSec: 2,
		StepRPS:         100,
		TestTimeSec:     10,
		ReportOptions: &ReportOptions{
			CSV: false,
			PNG: false,
		},
	}, &ControlAttackerMock{}, nil)
	r.controlled.Sleep = 1000
	maxRPS, _ := r.Run(context.TODO())
	// 100 attacker with 1 second blocked on request = 100 rps because clients are blocked
	require.GreaterOrEqual(t, int(maxRPS), rps)
}

func TestRunnerMaxRPSPrivateSystem(t *testing.T) {
	rps := 100
	r := NewRunner(&RunnerConfig{
		Name:            "test_runner",
		SystemMode:      PrivateSystem,
		Attackers:       20,
		AttackerTimeout: 1,
		StartRPS:        rps,
		StepDurationSec: 5,
		StepRPS:         1,
		TestTimeSec:     7,
		ReportOptions: &ReportOptions{
			CSV: false,
			PNG: false,
		},
	}, &ControlAttackerMock{}, nil)
	r.controlled.Sleep = 300
	maxRPS, err := r.Run(context.TODO())
	require.NoError(t, err)
	require.GreaterOrEqual(t, int(maxRPS), 69)
	require.Less(t, int(maxRPS), 74)
}

func TestRunnerMaxRPSOpenWorldSystem(t *testing.T) {
	r := NewRunner(&RunnerConfig{
		Name:            "test_runner",
		SystemMode:      OpenWorldSystem,
		AttackerTimeout: 1,
		StartRPS:        100,
		StepDurationSec: 5,
		StepRPS:         100,
		TestTimeSec:     17,
		ReportOptions: &ReportOptions{
			CSV: false,
			PNG: false,
		},
	}, &ControlAttackerMock{}, nil)
	maxRPS, err := r.Run(context.TODO())
	require.NoError(t, err)
	require.GreaterOrEqual(t, int(maxRPS), 400)
}

func TestRunnerConstantLoad(t *testing.T) {
	r := NewRunner(&RunnerConfig{
		Name:            "test_runner",
		SystemMode:      OpenWorldSystem,
		AttackerTimeout: 1,
		StartRPS:        30,
		TestTimeSec:     5,
		ReportOptions: &ReportOptions{
			CSV: false,
			PNG: false,
		},
	}, &ControlAttackerMock{}, nil)
	r.controlled.Sleep = 300
	maxRPS, err := r.Run(context.TODO())
	require.NoError(t, err)
	require.GreaterOrEqual(t, int(maxRPS), 30)
	require.Less(t, int(maxRPS), 33)

	r2 := NewRunner(&RunnerConfig{
		Name:            "test_runner",
		SystemMode:      PrivateSystem,
		Attackers:       300,
		AttackerTimeout: 1,
		StartRPS:        30,
		TestTimeSec:     5,
		ReportOptions: &ReportOptions{
			CSV: false,
			PNG: false,
		},
	}, &ControlAttackerMock{}, nil)
	r2.controlled.Sleep = 300
	maxRPS2, err2 := r2.Run(context.TODO())
	require.NoError(t, err2)
	require.Greater(t, int(maxRPS2), 30)
	require.Less(t, int(maxRPS2), 33)
}

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
		Attackers:       2000,
		AttackerTimeout: 1,
		StartRPS:        100,
		StepDurationSec: 5,
		StepRPS:         200,
		TestTimeSec:     10,
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
			SystemMode:      OpenWorldSystem,
			Attackers:       100,
			AttackerTimeout: 25,
			StartRPS:        10,
			StepDurationSec: 5,
			StepRPS:         10,
			TestTimeSec:     120,
			ReportOptions: &ReportOptions{
				CSV: true,
				PNG: true,
			},
		}, &ControlAttackerMock{}, nil)
		atomic.AddInt64(&r.controlled.Sleep, 10000)
		_, _ = r.Run(context.Background())
	}
}

func TestReportMetrics(t *testing.T) {
	r := NewRunner(&RunnerConfig{
		Name:            "test_runner",
		SystemMode:      PrivateSystem,
		Attackers:       10,
		AttackerTimeout: 1,
		StartRPS:        8,
		StepDurationSec: 5,
		StepRPS:         2,
		TestTimeSec:     10,
		ReportOptions: &ReportOptions{
			CSV: true,
			PNG: true,
		},
	}, &ControlAttackerMock{}, nil)
	r.controlled.Sleep = 500
	_, err := r.Run(context.TODO())
	require.NoError(t, err)
}
