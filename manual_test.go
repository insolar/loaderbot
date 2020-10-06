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
	"go.uber.org/goleak"
)

func TestManualDynamicLatencyHTTP(t *testing.T) {
	srv := RunTestServer("0.0.0.0:9031", 3000*time.Millisecond)
	// nolint
	defer srv.Shutdown(context.Background())
	for i := 0; i < 10; i++ {
		r := NewRunner(&RunnerConfig{
			TargetUrl:       "http://0.0.0.0:9031/json_body",
			Name:            "test_runner",
			SystemMode:      BoundRPS,
			Attackers:       1000,
			AttackerTimeout: 5,
			StartRPS:        1000,
			StepDurationSec: 5,
			StepRPS:         1000,
			TestTimeSec:     20,
			SuccessRatio:    1,

			ReportOptions: &ReportOptions{
				CSV: true,
				PNG: true,
			},
		}, &HTTPAttackerExample{}, nil)
		_, _ = r.Run(context.TODO())
	}
}

func TestManualDynamicLatencySync(t *testing.T) {
	r := NewRunner(&RunnerConfig{
		Name:            "test_runner",
		SystemMode:      BoundRPS,
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

func TestManualCorrectTickMetrics(t *testing.T) {
	r3 := NewRunner(&RunnerConfig{
		Name:            "test_runner_private_decrease",
		SystemMode:      BoundRPS,
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
		SystemMode:      BoundRPS,
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

func TestManualLeak(t *testing.T) {
	for i := 0; i < 10; i++ {
		r := NewRunner(&RunnerConfig{
			Name:            "test_runner_open_world_decrease",
			SystemMode:      BoundRPS,
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

func TestManualRunnerNginxStaticAttackFastHTTP(t *testing.T) {
	// go pprofTrace("fast_http", 40)
	// go tool trace -http=':8081' ${FILENAME}
	r := NewRunner(&RunnerConfig{
		TargetUrl:       "http://localhost:8080/static.html",
		Name:            "nginx_test",
		SystemMode:      BoundRPS,
		Attackers:       5000,
		AttackerTimeout: 25,
		StartRPS:        5000,
		StepDurationSec: 5,
		StepRPS:         3000,
		TestTimeSec:     40,
		SuccessRatio:    1,
	}, &FastHTTPAttackerExample{}, nil)
	_, _ = r.Run(context.TODO())
}

func TestManualRunnerNginxStaticAttackDefaultHTTP(t *testing.T) {
	// go pprofTrace("default_http", 40)
	// go tool trace -http=':8081' ${FILENAME}
	r := NewRunner(&RunnerConfig{
		TargetUrl:       "http://127.0.0.1:8080/static.html",
		Name:            "nginx_test",
		SystemMode:      BoundRPS,
		Attackers:       3000,
		AttackerTimeout: 25,
		StartRPS:        3000,
		StepDurationSec: 5,
		StepRPS:         2000,
		TestTimeSec:     40,
		SuccessRatio:    1,
	}, &HTTPAttackerExample{}, nil)
	_, _ = r.Run(context.TODO())
}

func TestManualRunnerRealServiceAttack(t *testing.T) {
	r := NewRunner(&RunnerConfig{
		TargetUrl:       "https://clients5.google.com/pagead/drt/dn/",
		Name:            "test_runner",
		SystemMode:      BoundRPS,
		Attackers:       3000,
		AttackerTimeout: 5,
		StartRPS:        1000,
		StepDurationSec: 5,
		StepRPS:         3000,
		TestTimeSec:     60,
	}, &HTTPAttackerExample{}, nil)
	_, _ = r.Run(context.TODO())
}

func TestManualPrometheus(t *testing.T) {
	srv := RunTestServer("0.0.0.0:9031", 50*time.Millisecond)
	// nolint
	defer srv.Shutdown(context.Background())
	time.Sleep(1 * time.Second)
	// go pprofTrace("default_http", 30)
	// sockets for test
	// sudo lsof -n -i | grep -e LISTEN -e ESTABLISHED | grep "___TestPr" | wc -l
	r := NewRunner(&RunnerConfig{
		TargetUrl:       "http://127.0.0.1:9031/json_body",
		Name:            "nginx_test",
		SystemMode:      BoundRPS,
		Attackers:       5000,
		AttackerTimeout: 25,
		StartRPS:        10,
		TestTimeSec:     120,
		SuccessRatio:    0.95,
		Prometheus:      &Prometheus{Enable: true},
	}, &HTTPAttackerExample{}, nil)
	_, _ = r.Run(context.TODO())
}

func TestManualHTTPLeak(t *testing.T) {
	defer goleak.VerifyNone(t)
	srv := RunTestServer("0.0.0.0:9031", 3000*time.Millisecond)
	// nolint
	defer srv.Shutdown(context.Background())
	cfg := &RunnerConfig{
		TargetUrl:       "http://0.0.0.0:9031/json_body",
		Name:            "test_runner",
		Attackers:       10,
		AttackerTimeout: 5,
		StartRPS:        10,
		TestTimeSec:     4,
	}
	{
		cfg.SystemMode = BoundRPS
		r := NewRunner(cfg, &HTTPAttackerExample{}, nil)
		_, err := r.Run(context.TODO())
		require.NoError(t, err)
	}
	{
		cfg.SystemMode = BoundRPS
		cfg.AttackerTimeout = 2
		r := NewRunner(cfg, &HTTPAttackerExample{}, nil)
		_, err := r.Run(context.TODO())
		require.NoError(t, err)
	}
}

func TestManualUnboundRPS(t *testing.T) {
	srv := RunTestServer("0.0.0.0:9031", 50*time.Millisecond)
	// nolint
	defer srv.Shutdown(context.Background())
	time.Sleep(1 * time.Second)
	r := NewRunner(&RunnerConfig{
		TargetUrl:       "http://127.0.0.1:9031/json_body",
		Name:            "nginx_test",
		SystemMode:      UnboundRPS,
		Attackers:       2,
		AttackerTimeout: 25,
		TestTimeSec:     300,
		SuccessRatio:    0.95,
		Prometheus:      &Prometheus{Enable: true},
	}, &HTTPAttackerExample{}, nil)
	_, _ = r.Run(context.TODO())
}
