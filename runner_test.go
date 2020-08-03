/*
 * // Copyright 2020 Insolar Network Ltd.
 * // All rights reserved.
 * // This material is licensed under the Insolar License version 1.0,
 * // available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.
 */

package loaderbot

import (
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
	_, err := r.Run()
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
	_, err := r.Run()
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
	_, err := r.Run()
	require.NoError(t, err)
	r2 := NewRunner(cfg, &ControlAttackerMock{}, nil)
	_, err2 := r2.Run()
	require.NoError(t, err2)
	cfg.SystemMode = OpenWorldSystem
	r3 := NewRunner(cfg, &ControlAttackerMock{}, nil)
	_, err3 := r3.Run()
	require.NoError(t, err3)
	r4 := NewRunner(cfg, &ControlAttackerMock{}, nil)
	_, err4 := r4.Run()
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
	_, _ = r.Run()
	require.Equal(t, true, r.Failed)
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
	_, err := r.Run()
	require.NoError(t, err)
	require.Empty(t, r.uniqErrors)
}

func TestPrivateSystemRunnerIsSync(t *testing.T) {
	rps := 100
	r := NewRunner(&RunnerConfig{
		Name:            "test_runner",
		SystemMode:      PrivateSystem,
		Attackers:       1,
		AttackerTimeout: 1,
		StartRPS:        rps,
		StepDurationSec: 5,
		StepRPS:         1,
		TestTimeSec:     5,
		ReportOptions: &ReportOptions{
			CSV: false,
			PNG: false,
		},
	}, &ControlAttackerMock{}, nil)
	r.controlled.Sleep = 30
	maxRPS, _ := r.Run()
	require.Less(t, int(maxRPS), rps)
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
	maxRPS, err := r.Run()
	require.NoError(t, err)
	require.GreaterOrEqual(t, int(maxRPS), 66)
	require.Less(t, int(maxRPS), 70)
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
	maxRPS, err := r.Run()
	require.NoError(t, err)
	require.GreaterOrEqual(t, int(maxRPS), 400)
}

func TestRunnerConstantLoad(t *testing.T) {
	r := NewRunner(&RunnerConfig{
		Name:            "test_runner",
		SystemMode:      OpenWorldSystem,
		AttackerTimeout: 1,
		StartRPS:        30,
		TestTimeSec:     15,
		ReportOptions: &ReportOptions{
			CSV: false,
			PNG: false,
		},
	}, &ControlAttackerMock{}, nil)
	r.controlled.Sleep = 300
	maxRPS, err := r.Run()
	require.NoError(t, err)
	require.Greater(t, int(maxRPS), 30)
	require.Less(t, int(maxRPS), 33)

	r2 := NewRunner(&RunnerConfig{
		Name:            "test_runner",
		SystemMode:      PrivateSystem,
		Attackers:       300,
		AttackerTimeout: 1,
		StartRPS:        30,
		TestTimeSec:     15,
		ReportOptions: &ReportOptions{
			CSV: false,
			PNG: false,
		},
	}, &ControlAttackerMock{}, nil)
	r2.controlled.Sleep = 300
	maxRPS2, err2 := r2.Run()
	require.NoError(t, err2)
	require.Greater(t, int(maxRPS2), 30)
	require.Less(t, int(maxRPS2), 33)
}

func TestDynamicLatency(t *testing.T) {
	t.Skip("only manual run")
	r := NewRunner(&RunnerConfig{
		Name:            "test_runner",
		SystemMode:      OpenWorldSystem,
		Attackers:       1000,
		AttackerTimeout: 25,
		StartRPS:        100,
		StepDurationSec: 5,
		StepRPS:         100,
		TestTimeSec:     60,
		ReportOptions: &ReportOptions{
			CSV: true,
			PNG: true,
		},
	}, &ControlAttackerMock{}, nil)
	r.controlled.Sleep = 100
	latCfg := ServiceLatencyChangeConfig{
		R:             r,
		Interval:      1 * time.Second,
		LatencyStepMs: 300,
		Times:         30,
		LatencyFlag:   increaseLatency,
	}
	changeAttackersLatency(latCfg)
	_, _ = r.Run()
}

func TestRunnerRealServiceAttack(t *testing.T) {
	t.Skip("only manual run")
	r := NewRunner(&RunnerConfig{
		TargetUrl:       "https://clients5.google.com/pagead/drt/dn/",
		Name:            "test_runner",
		SystemMode:      OpenWorldSystem,
		Attackers:       1,
		AttackerTimeout: 1,
		StartRPS:        100,
		StepDurationSec: 5,
		StepRPS:         200,
		TestTimeSec:     30,
	}, &HTTPAttackerExample{}, nil)
	_, _ = r.Run()
}
