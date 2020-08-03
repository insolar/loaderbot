package loaderbot

import (
	"testing"
	"time"
)

const (
	address = "localhost:50051"
)

func TestServiceStream(t *testing.T) {
	s := RunService(address)
	defer s.GracefulStop()
	time.Sleep(1 * time.Second)
	c := NewNodeClient(address)
	ch := make(chan []AttackResult)
	c.StartRunner(RunnerConfig{
		TargetUrl:       "https://clients5.google.com/pagead/drt/dn/",
		Name:            "test_runner",
		SystemMode:      OpenWorldSystem,
		Attackers:       1,
		AttackerTimeout: 1,
		StartRPS:        100,
		StepDurationSec: 1,
		StepRPS:         200,
		TestTimeSec:     5,
		ReportOptions: &ReportOptions{
			Stream: true,
		},
	}, ch)
}

func TestClusterClient(t *testing.T) {
	s1 := RunService(address)
	defer s1.GracefulStop()
	s2 := RunService("localhost:50052")
	defer s2.GracefulStop()
	time.Sleep(1 * time.Second)
	c := NewClusterClient()
	c.RunClusterTestFromConfig(RunnerConfig{
		TargetUrl:       "https://clients5.google.com/pagead/drt/dn/",
		Name:            "test_runner",
		SystemMode:      OpenWorldSystem,
		Attackers:       1,
		AttackerTimeout: 1,
		StartRPS:        100,
		StepDurationSec: 1,
		StepRPS:         200,
		TestTimeSec:     5,
		ReportOptions: &ReportOptions{
			Stream: true,
		},
	})
}
