#### Loaderbot
Minimalistic load tool

Implement attacker
```
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
```
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
 when amount of attackers is unknown, but you still have RPS requirements
```
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
If you need some data to be shared between attackers, here an example
```
	cfg := &loaderbot.RunnerConfig{
		TargetUrl:        target,
		Name:             "get_attack",
		Attackers:        500,
		AttackerTimeout:  25,
		StartRPS:         440,
		StepDurationSec:  20,
		StepRPS:          10,
		TestTimeSec:      1200,
		DynamicAttackers: true,
		ScalingAttackers: 200,
		ScalingSkipTicks: 1,
		FailOnFirstError: true,
	}
	lt := loaderbot.NewRunner(cfg,
		&ve_perf_tests.GetContractTestAttack{},
		&loaderbot.TestData{
			Mutex: &sync.Mutex{},
			Data:  wallets,
		},
	)
	maxRPS, _ := lt.Run()
```

#### Development
```
make test
make lint
```