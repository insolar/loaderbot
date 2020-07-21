package loaderbot

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestPrivateSystemRunnerSuccess(t *testing.T) {
	r := NewRunner(&RunnerConfig{
		Name:            "",
		Attackers:       10,
		AttackerTimeout: 1,
		StartRPS:        8,
		StepDurationSec: 5,
		StepRPS:         2,
		TestTimeSec:     10,
	}, &ControlAttackerMock{}, nil)
	_, err := r.Run()
	require.NoError(t, err)
}

func TestOpenWorldSystemRunnerSuccess(t *testing.T) {
	r := NewRunner(&RunnerConfig{
		Name:            "",
		SystemMode:      OpenWorldSystem,
		Attackers:       300,
		AttackerTimeout: 1,
		StartRPS:        100,
		StepDurationSec: 10,
		StepRPS:         20,
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
		AttackersAmount: 3,
	}
	withControllableAttackers(cfg)
	serviceErrorAfter(serviceError, 1*time.Nanosecond)
	_, _ = r.Run()
	require.Equal(t, true, r.Failed)
}

func TestPrivateSystemRunnerIsSync(t *testing.T) {
	rps := 100
	r := NewRunner(&RunnerConfig{
		Name:            "",
		SystemMode:      PrivateSystem,
		Attackers:       1,
		AttackerTimeout: 1,
		StartRPS:        rps,
		StepDurationSec: 5,
		StepRPS:         1,
		TestTimeSec:     5,
	}, &ControlAttackerMock{}, nil)

	// decrease mock service latency so clients is blocked
	r.controlled.Sleep = 30
	_, _ = r.Run()
	r.metricsMu.Lock()
	defer r.metricsMu.Unlock()
	for _, m := range r.stepMetrics {
		require.Less(t, int(m.Rate), rps)
	}
}

func TestRunnerMaxRPSPrivateSystem(t *testing.T) {
	rps := 100
	r := NewRunner(&RunnerConfig{
		Name:            "",
		SystemMode:      PrivateSystem,
		Attackers:       1,
		AttackerTimeout: 1,
		StartRPS:        rps,
		StepDurationSec: 5,
		StepRPS:         1,
		TestTimeSec:     5,
	}, &ControlAttackerMock{}, nil)
	r.controlled.Sleep = 300
	maxRPS, err := r.Run()
	require.NoError(t, err)
	require.Equal(t, int(maxRPS), 3)
}

func TestRunnerMaxRPSOpenWorldSystem(t *testing.T) {
	r := NewRunner(&RunnerConfig{
		Name:            "",
		SystemMode:      OpenWorldSystem,
		AttackerTimeout: 1,
		StartRPS:        100,
		StepDurationSec: 5,
		StepRPS:         100,
		TestTimeSec:     15,
	}, &ControlAttackerMock{}, nil)
	maxRPS, err := r.Run()
	require.NoError(t, err)
	require.Equal(t, 300, int(maxRPS))
}

func TestDynamicLatency(t *testing.T) {
	t.Skip("only manual run")
	r := NewRunner(&RunnerConfig{
		Name:            "",
		SystemMode:      OpenWorldSystem,
		Attackers:       500,
		AttackerTimeout: 25,
		StartRPS:        100,
		StepDurationSec: 2,
		StepRPS:         10,
		TestTimeSec:     100,
	}, &ControlAttackerMock{}, nil)
	r.controlled.Sleep = 30
	// latCfg := ServiceLatencyChangeConfig{
	// 	R:             r,
	// 	Interval:      1 * time.Second,
	// 	LatencyStepMs: 300,
	// 	Times:         30,
	// 	LatencyFlag:   increaseLatency,
	// }
	// changeAttackersLatency(latCfg)
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
		StartRPS:        10,
		StepDurationSec: 5,
		StepRPS:         50,
		TestTimeSec:     20,
	}, &HTTPAttackerExample{}, nil)
	_, _ = r.Run()
}
