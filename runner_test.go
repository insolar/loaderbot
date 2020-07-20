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
	cfg := ControllableConfig{
		R:               r,
		ControlChan:     serviceError,
		AttackerLatency: 100 * time.Millisecond,
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
		TestTimeSec:      60,
	}, &ControlAttackerMock{}, nil)
	cfg := ControllableConfig{
		R:               r,
		ControlChan:     make(chan bool),
		AttackerLatency: 10 * time.Millisecond,
		AttackersAmount: 3,
	}
	withControllableAttackers(cfg)
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
		TestTimeSec:      60,
	}, &ControlAttackerMock{}, nil)
	cfg := ControllableConfig{
		R:               r,
		ControlChan:     make(chan bool),
		AttackerLatency: 900 * time.Millisecond,
		AttackersAmount: 300,
	}
	withControllableAttackers(cfg)
	latCfg := ServiceLatencyChangeConfig{
		R:             r,
		Interval:      1 * time.Second,
		LatencyStepMs: 30,
		Times:         10,
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
