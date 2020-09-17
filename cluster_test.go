// +build !race
// by default we are using single HTTP client, so in cluster mode client is one per vm, in tests for multiple nodes that will race
// otherwise, when creating multiple transports default client will leak goroutines!

package loaderbot

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestCommonClusterClient(t *testing.T) {
	s1 := RunService("localhost:50051")
	defer s1.GracefulStop()
	s2 := RunService("localhost:50052")
	defer s2.GracefulStop()
	s3 := RunService("localhost:50053")
	defer s3.GracefulStop()
	s4 := RunService("localhost:50054")
	defer s4.GracefulStop()
	time.Sleep(1 * time.Second)
	c := NewClusterClient(&RunnerConfig{
		TargetUrl:       "https://clients5.google.com/pagead/drt/dn/",
		Name:            "test_runner",
		SystemMode:      BoundRPS,
		InstanceType:    "HTTPAttackerExample",
		Attackers:       100,
		AttackerTimeout: 1,
		StartRPS:        100,
		StepDurationSec: 5,
		StepRPS:         20,
		TestTimeSec:     10,
		LogEncoding:     "console",
		LogLevel:        "info",
		ReportOptions: &ReportOptions{
			CSV:    true,
			PNG:    true,
			Stream: true,
		},
		ClusterOptions: &ClusterOptions{
			Nodes: []string{"localhost:50051", "localhost:50052", "localhost:50053", "localhost:50054"},
		},
	})
	c.Run()
	require.Equal(t, false, c.failed)
}

func TestCommonClusterShutdownOnError(t *testing.T) {
	s1 := RunService("localhost:50055")
	defer s1.GracefulStop()
	s2 := RunService("localhost:50056")
	defer s2.GracefulStop()
	time.Sleep(1 * time.Second)
	c := NewClusterClient(&RunnerConfig{
		TargetUrl:       "",
		Name:            "test_runner",
		InstanceType:    "HTTPAttackerExample",
		SystemMode:      BoundRPS,
		Attackers:       10,
		AttackerTimeout: 1,
		StartRPS:        10,
		StepDurationSec: 2,
		StepRPS:         20,
		TestTimeSec:     6,
		SuccessRatio:    1,
		LogEncoding:     "console",
		LogLevel:        "info",
		ReportOptions: &ReportOptions{
			Stream: true,
		},
		ClusterOptions: &ClusterOptions{
			Nodes: []string{"localhost:50055", "localhost:50056"},
		},
	})
	c.Run()
	require.Equal(t, true, c.failed)
}

func TestCommonClusterNodeIsBusy(t *testing.T) {
	s1 := RunService("localhost:50057")
	defer s1.GracefulStop()
	time.Sleep(1 * time.Second)

	cfg := &RunnerConfig{
		TargetUrl:       "https://clients5.google.com/pagead/drt/dn/",
		Name:            "test_runner",
		InstanceType:    "HTTPAttackerExample",
		SystemMode:      BoundRPS,
		Attackers:       10,
		AttackerTimeout: 1,
		StartRPS:        100,
		StepDurationSec: 5,
		StepRPS:         20,
		TestTimeSec:     2,
		LogEncoding:     "console",
		LogLevel:        "info",
		ReportOptions: &ReportOptions{
			CSV:    true,
			PNG:    true,
			Stream: true,
		},
		ClusterOptions: &ClusterOptions{
			Nodes: []string{"localhost:50057"},
		},
	}
	c := NewClusterClient(cfg)
	go c.Run()
	time.Sleep(1 * time.Second)
	c2 := NewClusterClient(cfg)
	require.True(t, c2.failed)
}
