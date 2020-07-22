#### Loaderbot
Minimalistic load tool

#### How to use

Implement attacker
```go
type HTTPAttackerExample struct {
	*Runner
	client *http.Client
}

func (a *HTTPAttackerExample) Clone(r *Runner) Attack {
	return &HTTPAttackerExample{Runner: r}
}

func (a *HTTPAttackerExample) Setup(c RunnerConfig) error {
    // setup any client
	a.client = loaderbot.NewLoggingHTTPClient(c.DumpTransport, 10)
	return nil
}

func (a *HTTPAttackerExample) Do(_ context.Context) DoResult {
	_, err := a.client.Get(a.cfg.TargetUrl)
	return DoResult{
		RequestLabel: a.name,
		Error:        err,
	}
}

func (a *HTTPAttackerExample) Teardown() error {
	return nil
}
```
Run test with sync attackers when system is "closed" type, when response time increases,
 attackers may be blocked and it's okay for this mode
```go
cfg := &loaderbot.RunnerConfig{
		TargetUrl:       "https://clients5.google.com/pagead/drt/dn/",
		Name:            "abc",
		SystemMode:      loaderbot.PrivateSystem,
		Attackers:       100,
		AttackerTimeout: 5,
		StartRPS:        100,
		StepDurationSec: 10,
		StepRPS:         300,
		TestTimeSec:     200,
	}
lt := loaderbot.NewRunner(cfg, &loaderbot.HTTPAttackerExample{}, nil)
maxRPS, _ := lt.Run()
```
or with async attackers, when your system is "open" type, ex. search engine,
 when amount of attackers, and their identity is unknown, but you still have RPS requirements
```go
cfg := &loaderbot.RunnerConfig{
		TargetUrl:       "https://clients5.google.com/pagead/drt/dn/",
		Name:            "abc",
		SystemMode:      loaderbot.OpenWorldSystem,
		AttackerTimeout: 5,
		StartRPS:        100,
		StepDurationSec: 30,
		StepRPS:         10,
		TestTimeSec:     200,
	}
lt := loaderbot.NewRunner(cfg, &loaderbot.HTTPAttackerExample{}, nil)
maxRPS, _ := lt.Run()
```
see more [examples](examples/tests)

Config options
```go
// RunnerConfig runner configuration
type RunnerConfig struct {
	// TargetUrl target base url
	TargetUrl string
	// Name of a runner instance
	Name string
	// SystemMode PrivateSystem | OpenWorldSystem
	// PrivateSystem:
	// if application under test is a private system sync runner attackers will wait for response
	// in case your system is private and you know how many sync clients can act
	// OpenWorldSystem:
	// if application under test is an open world system async runner attackers will fire requests without waiting
	// it creates some inaccuracy in results, so you can check latencies using service metrics to be precise,
	// but the test will be more realistic from clients point of view
	SystemMode SystemMode
	// Attackers constant amount of attackers,
	// if SystemMode is "OpenWorldSystem", attackers will be spawn on demand to meet rps
	Attackers int
	// AttackerTimeout timeout of attacker
	AttackerTimeout int
	// StartRPS initial requests per seconds rate
	StartRPS int
	// StepDurationSec duration of step in which rps is increased by StepRPS
	StepDurationSec int
	// StepRPS amount of requests per second which will be added in next step,
	// if StepRPS = 0 rate is constant, default StepDurationSec is 30 sec is applied,
	// just to keep 30s aggregation metrics
	StepRPS int
	// TestTimeSec test timeout
	TestTimeSec int
	// WaitBeforeSec time to wait before start in case we didn't know start criteria
	WaitBeforeSec int
	// Dumptransport dump http requests to stdout
	DumpTransport bool
	// GoroutinesDump dumps goroutines stack for debug purposes
	GoroutinesDump bool
	// FailOnFirstError fails test on first error
	FailOnFirstError bool
	// LogLevel debug|info, etc.
	LogLevel string
	// LogEncoding json|console
	LogEncoding string
}
```

#### Development
```
make test
make lint
```