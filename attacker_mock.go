package loaderbot

import (
	"context"
	"time"
)

type Attacker struct {
	name int
	r    *Runner
	l    *Logger
}

func (a *Attacker) Setup(c RunnerConfig) error {
	panic("implement me")
}

func (a *Attacker) Do(_ context.Context) DoResult {
	time.Sleep(time.Duration(50) * time.Millisecond)
	return DoResult{RequestLabel: a.r.name}
}

func (a *Attacker) Teardown() error {
	return nil
}

func (a *Attacker) Clone(r *Runner) Attack {
	return a
}

func NewAttacker(name int, r *Runner) *Attacker {
	l := r.L.Clone()
	ll := l.With("attacker", name)
	return &Attacker{name, r, ll}
}
