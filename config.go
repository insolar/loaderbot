package loaderbot

// RunnerConfig runner configuration
type RunnerConfig struct {
	// Name of a runner instance
	Name string
	// Attackers constant amount of attackers
	Attackers int
	// AttackTimeout timeout of attacker
	AttackTimeout int
	// StartRPS start amount of requests per second
	StartRPS int
	// EndRPS target amount of requests per second
	EndRPS int
	// StepDuration duration of step in which rps is increased by StepRPS
	StepDuration int
	// StepRPS amount of requests per second which will be added in next step
	StepRPS int
	// Timeout test timeout
	Timeout int
	// WaitBefore time to wait before start in case we didn't know start criteria
	WaitBefore int
}
