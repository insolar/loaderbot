package loaderbot

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestRunnerSuccess(t *testing.T) {
	r := NewRunner(&RunnerConfig{
		Name:            "",
		Attackers:       1,
		AttackerTimeout: 1,
		StartRPS:        1,
		StepDurationSec: 5,
		StepRPS:         2,
		TestTimeSec:     5,
	}, &ControlAttackerMock{}, nil)
	_, err := r.Run()
	require.NoError(t, err)
}

func TestMultipleRunnersSuccess(t *testing.T) {
	cfg := &RunnerConfig{
		Name:            "",
		Attackers:       1,
		AttackerTimeout: 1,
		StartRPS:        1,
		StepDurationSec: 5,
		StepRPS:         2,
		TestTimeSec:     1,
	}
	r := NewRunner(cfg, &ControlAttackerMock{}, nil)
	_, err := r.Run()
	require.NoError(t, err)
	r2 := NewRunner(cfg, &ControlAttackerMock{}, nil)
	_, err2 := r2.Run()
	require.NoError(t, err2)
}

func TestRunnerFailOnFirstError(t *testing.T) {
	r := NewRunner(&RunnerConfig{
		Name:             "",
		Attackers:        10,
		AttackerTimeout:  1,
		StartRPS:         1,
		StepDurationSec:  5,
		StepRPS:          2,
		TestTimeSec:      5,
		FailOnFirstError: true,
	}, &ControlAttackerMock{}, nil)
	serviceError := make(chan bool)
	r.controlled.Sleep = 80
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

func TestScalingWhenLatencyIncreases(t *testing.T) {
	r := NewRunner(&RunnerConfig{
		Name:             "",
		Attackers:        300,
		AttackerTimeout:  1,
		StartRPS:         100,
		StepDurationSec:  10,
		StepRPS:          50,
		DynamicAttackers: true,
		TestTimeSec:      40,
	}, &ControlAttackerMock{}, nil)
	r.controlled.Sleep = 30
	latCfg := ServiceLatencyChangeConfig{
		R:             r,
		Interval:      1 * time.Second,
		LatencyStepMs: 30,
		Times:         30,
		LatencyFlag:   increaseLatency,
	}
	changeAttackersLatency(latCfg)
	_, _ = r.Run()
	r.tickMetricsMu.Lock()
	defer r.tickMetricsMu.Unlock()
	for _, m := range r.tickUpdateMetrics {
		t.Log(m.Rate, m.TargetRate)
	}
	for k, v := range r.scalingInfo.TicksInSteps {
		t.Logf("step: %d, scaling ticks: %d", k, v)
	}
}

func TestNotScalingWhenLatencyDecreases(t *testing.T) {
	r := NewRunner(&RunnerConfig{
		Name:             "",
		Attackers:        300,
		AttackerTimeout:  1,
		StartRPS:         100,
		StepDurationSec:  10,
		StepRPS:          10,
		DynamicAttackers: true,
		TestTimeSec:      40,
	}, &ControlAttackerMock{}, nil)
	r.controlled.Sleep = 900
	latCfg := ServiceLatencyChangeConfig{
		R:             r,
		Interval:      1 * time.Second,
		LatencyStepMs: 30,
		Times:         30,
		LatencyFlag:   decreaseLatency,
	}
	changeAttackersLatency(latCfg)
	_, _ = r.Run()
	r.tickMetricsMu.Lock()
	defer r.tickMetricsMu.Unlock()
	for _, m := range r.tickUpdateMetrics {
		t.Log(m.Rate, m.TargetRate)
	}
	require.Empty(t, r.scalingInfo.TicksInSteps)
}

func TestRunnerRealServiceAttack(t *testing.T) {
	t.Skip("only manual run")
	r := NewRunner(&RunnerConfig{
		TargetUrl:        "https://clients5.google.com/pagead/drt/dn/",
		Name:             "test_runner",
		Attackers:        10,
		AttackerTimeout:  1,
		StartRPS:         10,
		StepDurationSec:  1,
		StepRPS:          50,
		TestTimeSec:      20,
		DynamicAttackers: true,
	}, &HTTPAttackerExample{}, nil)
	maxRPS, _ := r.Run()
	t.Logf("max rps: %.2f", maxRPS)
}
