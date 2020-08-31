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

	"github.com/stretchr/testify/require"
)

func TestCommonPrivateSystemRunnerSuccess(t *testing.T) {
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

func TestCommonOpenWorldSystemRunnerSuccess(t *testing.T) {
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

func TestCommonMultipleRunnersSuccess(t *testing.T) {
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

func TestCommonRunnerFailOnFirstError(t *testing.T) {
	r := NewRunner(&RunnerConfig{
		Name:            "test_runner",
		Attackers:       10,
		AttackerTimeout: 1,
		StartRPS:        1000,
		StepDurationSec: 5,
		StepRPS:         2,
		TestTimeSec:     5,
		SuccessRatio:    1,
		ReportOptions: &ReportOptions{
			CSV: false,
			PNG: false,
		},
	}, &ControlAttackerMock{}, nil)
	serviceError := make(chan bool)
	cfg := ControllableConfig{
		R:               r,
		ControlChan:     serviceError,
		AttackersAmount: 1000,
	}
	withControllableAttackers(cfg)
	serviceErrorAfter(serviceError, 3*time.Second)
	_, _ = r.Run(context.TODO())
	require.Equal(t, int64(1), r.Failed)
}

func TestCommonRunnerHangedRequestsAfterTimeoutNoError(t *testing.T) {
	r := NewRunner(&RunnerConfig{
		Name:            "test_runner",
		SystemMode:      PrivateSystem,
		Attackers:       1,
		AttackerTimeout: 5,
		StartRPS:        1,
		StepDurationSec: 5,
		StepRPS:         2,
		TestTimeSec:     2,
		SuccessRatio:    1,
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

func TestCommonPrivateSystemRunnerIsSync(t *testing.T) {
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

func TestCommonRunnerMaxRPSPrivateSystem(t *testing.T) {
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

func TestCommonRunnerMaxRPSOpenWorldSystem(t *testing.T) {
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

func TestCommonRunnerConstantLoad(t *testing.T) {
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
	require.GreaterOrEqual(t, int(maxRPS2), 30)
	require.LessOrEqual(t, int(maxRPS2), 33)
}

func TestCommonReportMetrics(t *testing.T) {
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
