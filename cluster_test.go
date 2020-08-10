package loaderbot

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestClusterClient(t *testing.T) {
	s1 := RunService("localhost:50051")
	defer s1.GracefulStop()
	s2 := RunService("localhost:50052")
	defer s2.GracefulStop()
	s3 := RunService("localhost:50053")
	defer s3.GracefulStop()
	time.Sleep(1 * time.Second)
	c := NewClusterClient(&RunnerConfig{
		TargetUrl:       "https://clients5.google.com/pagead/drt/dn/",
		Name:            "test_runner",
		SystemMode:      OpenWorldSystem,
		Attackers:       1,
		AttackerTimeout: 1,
		StartRPS:        10,
		StepDurationSec: 2,
		StepRPS:         20,
		TestTimeSec:     60,
		LogEncoding:     "console",
		LogLevel:        "info",
		ReportOptions: &ReportOptions{
			Stream: true,
		},
		ClusterOptions: &ClusterOptions{
			// Nodes: []string{"localhost:50051", "localhost:50052"},
			Nodes: []string{"localhost:50051", "localhost:50052", "localhost:50053"},
		},
	})
	c.Run()
	require.Equal(t, false, c.failed)
}

func TestClusterShutdownOnError(t *testing.T) {
	s1 := RunService("localhost:50053")
	defer s1.GracefulStop()
	s2 := RunService("localhost:50054")
	defer s2.GracefulStop()
	time.Sleep(1 * time.Second)
	c := NewClusterClient(&RunnerConfig{
		TargetUrl:        "",
		Name:             "test_runner",
		SystemMode:       OpenWorldSystem,
		Attackers:        1,
		AttackerTimeout:  1,
		StartRPS:         10,
		StepDurationSec:  2,
		StepRPS:          20,
		TestTimeSec:      6,
		FailOnFirstError: true,
		LogEncoding:      "console",
		LogLevel:         "info",
		ReportOptions: &ReportOptions{
			Stream: true,
		},
		ClusterOptions: &ClusterOptions{
			Nodes: []string{"localhost:50053", "localhost:50054"},
		},
	})
	c.Run()
	require.Equal(t, true, c.failed)
}
