/*
 * // Copyright 2020 Insolar Network Ltd.
 * // All rights reserved.
 * // This material is licensed under the Insolar License version 1.0,
 * // available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.
 */

package attackers

import (
	"context"
	"net/http"

	"github.com/insolar/loaderbot"
)

type AttackerExample struct {
	*loaderbot.Runner
	client *http.Client
}

func (a *AttackerExample) Clone(r *loaderbot.Runner) loaderbot.Attack {
	return &AttackerExample{Runner: r}
}

func (a *AttackerExample) Setup(c loaderbot.RunnerConfig) error {
	a.client = loaderbot.NewLoggingHTTPClient(c.DumpTransport, 10)
	return nil
}

func (a *AttackerExample) Do(_ context.Context) loaderbot.DoResult {
	_, err := a.client.Get(a.Cfg.TargetUrl)
	return loaderbot.DoResult{
		RequestLabel: a.Name,
		Error:        err,
	}
}

func (a *AttackerExample) Teardown() error {
	return nil
}
