package main

import "github.com/insolar/loaderbot"

func main() {
	cfg := &loaderbot.RunnerConfig{
		Name:          "abc",
		Attackers:     2000,
		AttackTimeout: 5,
		StartRPS:      20,
		EndRPS:        100,
		StepDuration:  5,
		StepRPS:       5,
		Timeout:       200,
		WaitBefore:    10,
	}
	lt := loaderbot.NewRunner(cfg)
	lt.Run()
}
