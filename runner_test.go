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

func TestCommonBoundRPSRunnerSuccess(t *testing.T) {
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
		SystemMode:      BoundRPS,
		Attackers:       10,
		AttackerTimeout: 5,
		StartRPS:        1,
		StepDurationSec: 5,
		StepRPS:         2,
		TestTimeSec:     2,
		SuccessRatio:    1,
		ReportOptions: &ReportOptions{
			CSV: true,
		},
	}, &ControlAttackerMock{}, nil)
	// request still hangs when the test ends, but it's not an error because test has ended
	r.controlled.Sleep = 5000
	_, err := r.Run(context.TODO())
	require.NoError(t, err)
	require.Empty(t, r.uniqErrors)
}

func TestCommonBoundRPSRunnerIsSync(t *testing.T) {
	rps := 100
	r := NewRunner(&RunnerConfig{
		Name:            "test_runner",
		SystemMode:      BoundRPS,
		Attackers:       100,
		AttackerTimeout: 5,
		StartRPS:        rps,
		StepDurationSec: 2,
		StepRPS:         100,
		TestTimeSec:     10,
		ReportOptions: &ReportOptions{
			CSV: true,
		},
	}, &ControlAttackerMock{}, nil)
	r.controlled.Sleep = 1000
	maxRPS, _ := r.Run(context.TODO())
	// 100 attacker with 1 second blocked on request = 100 rps because clients are blocked
	require.GreaterOrEqual(t, int(maxRPS), rps)
}

func TestCommonRunnerMaxRPSBoundRPS(t *testing.T) {
	rps := 100
	r := NewRunner(&RunnerConfig{
		Name:            "test_runner",
		SystemMode:      BoundRPS,
		Attackers:       20,
		AttackerTimeout: 1,
		StartRPS:        rps,
		TestTimeSec:     7,
		ReportOptions: &ReportOptions{
			CSV: true,
		},
	}, &ControlAttackerMock{}, nil)
	r.controlled.Sleep = 300
	maxRPS, err := r.Run(context.TODO())
	require.NoError(t, err)
	require.GreaterOrEqual(t, int(maxRPS), 69)
	require.LessOrEqual(t, int(maxRPS), 74)
}

func TestCommonRunnerConstantLoad(t *testing.T) {
	r := NewRunner(&RunnerConfig{
		Name:            "test_runner",
		SystemMode:      BoundRPS,
		Attackers:       100,
		AttackerTimeout: 1,
		StartRPS:        30,
		TestTimeSec:     5,
	}, &ControlAttackerMock{}, nil)
	r.controlled.Sleep = 300
	maxRPS, err := r.Run(context.TODO())
	require.NoError(t, err)
	require.GreaterOrEqual(t, int(maxRPS), 30)
	require.Less(t, int(maxRPS), 33)
}

func TestCommonReportMetrics(t *testing.T) {
	r := NewRunner(&RunnerConfig{
		Name:            "test_runner",
		SystemMode:      BoundRPS,
		Attackers:       10,
		AttackerTimeout: 1,
		StartRPS:        8,
		StepDurationSec: 5,
		StepRPS:         2,
		TestTimeSec:     10,
		ReportOptions: &ReportOptions{
			HTMLDir: "test_html",
			CSVDir:  "test_csv",
			CSV:     true,
			PNG:     true,
			Stream:  false,
		},
	}, &ControlAttackerMock{}, nil)
	r.controlled.Sleep = 500
	_, err := r.Run(context.TODO())
	require.NoError(t, err)
}

func TestCommonGracefulPrometheusMultipleRunners(t *testing.T) {
	cfg := &RunnerConfig{
		Name:            "test_runner",
		SystemMode:      BoundRPS,
		Attackers:       10,
		AttackerTimeout: 1,
		StartRPS:        8,
		StepDurationSec: 5,
		StepRPS:         2,
		TestTimeSec:     2,
		Prometheus: &Prometheus{
			Enable: true,
		},
	}
	r := NewRunner(cfg, &ControlAttackerMock{}, nil)
	_, err := r.Run(context.TODO())
	require.NoError(t, err)
	r2 := NewRunner(cfg, &ControlAttackerMock{}, nil)
	_, err2 := r2.Run(context.TODO())
	require.NoError(t, err2)
}

func TestCommonTypedInstance(t *testing.T) {
	cfg := &RunnerConfig{
		Name:            "test_runner",
		SystemMode:      BoundRPS,
		Attackers:       10,
		AttackerTimeout: 1,
		StartRPS:        8,
		StepDurationSec: 5,
		StepRPS:         2,
		TestTimeSec:     2,
		Prometheus: &Prometheus{
			Enable: true,
		},
	}
	r := NewRunner(cfg, AttackerFromString("TypedAttackerMock1"), nil)
	_, err := r.Run(context.TODO())
	require.NoError(t, err)
}

func TestCommonAutoscale(t *testing.T) {
	rps := 100
	r := NewRunner(&RunnerConfig{
		Name:            "test_runner",
		SystemMode:      BoundRPSAutoscale,
		Attackers:       100,
		AttackerTimeout: 5,
		StartRPS:        rps,
		StepDurationSec: 2,
		StepRPS:         100,
		TestTimeSec:     10,
		ReportOptions: &ReportOptions{
			CSV: true,
		},
	}, &ControlAttackerMock{}, nil)
	r.controlled.Sleep = 1000
	maxRPS, _ := r.Run(context.TODO())
	// 100 attacker with 1 second blocked on request = 100 rps because clients are blocked
	require.GreaterOrEqual(t, int(maxRPS), rps)
}
