package loaderbot

// RunnerConfig runner configuration
type RunnerConfig struct {
	// TargetUrl target base url
	TargetUrl string
	// Name of a runner instance
	Name string
	// Attackers constant amount of attackers
	Attackers int
	// AttackerTimeout timeout of attacker
	AttackerTimeout int
	// StartRPS start amount of requests per second
	StartRPS int
	// StepDurationSec duration of step in which rps is increased by StepRPS
	StepDurationSec int
	// StepRPS amount of requests per second which will be added in next step
	StepRPS int
	// TestTimeSec test timeout
	TestTimeSec int
	// WaitBeforeSec time to wait before start in case we didn't know start criteria
	WaitBeforeSec int
	// Dumptransport dump http requests to stdout
	DumpTransport bool
	// GoroutinesDump
	GoroutinesDump bool
	// Dynamic creates attackers if current rps < target rps
	DynamicAttackers bool
	// FailOnFirstError fails on first error
	FailOnFirstError bool
	// LogLevel debug|info, etc.
	LogLevel string
	// LogEncoding json|console
	LogEncoding string
}

// Validate checks all settings and returns a list of strings with problems.
func (c RunnerConfig) Validate() (list []string) {
	if c.Attackers <= 0 {
		list = append(list, "please set attackers > 0")
	}
	if c.AttackerTimeout <= 0 {
		list = append(list, "please set attacker timeout > 0, seconds")
	}
	if c.StartRPS <= 0 {
		list = append(list, "please set start rps > 0")
	}
	if c.StepDurationSec <= 0 {
		list = append(list, "please set step duration > 0, seconds")
	}
	if c.StepRPS <= 0 {
		list = append(list, "please set end rps > 0")
	}
	if c.TestTimeSec <= 0 {
		list = append(list, "please set test time rps > 0, seconds")
	}
	return
}
