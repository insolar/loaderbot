/*
 * // Copyright 2020 Insolar Network Ltd.
 * // All rights reserved.
 * // This material is licensed under the Insolar License version 1.0,
 * // available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.
 */

package loaderbot

import (
	"context"
	"io"
	"io/ioutil"
	"net/http"
)

func init() {
	RegisterAttacker("HTTPAttackerExample", &HTTPAttackerExample{})
}

type HTTPAttackerExample struct {
	*Runner
}

func (a *HTTPAttackerExample) Clone(r *Runner) Attack {
	return &HTTPAttackerExample{Runner: r}
}

func (a *HTTPAttackerExample) Setup(c RunnerConfig) error {
	return nil
}

func (a *HTTPAttackerExample) Do(ctx context.Context) DoResult {
	req, _ := http.NewRequestWithContext(ctx, "GET", a.Cfg.TargetUrl, nil)
	res, err := a.HTTPClient.Do(req)
	if res != nil {
		if _, err = io.Copy(ioutil.Discard, res.Body); err != nil {
			return DoResult{
				RequestLabel: a.Name,
				Error:        err.Error(),
			}
		}
		defer res.Body.Close()
	}
	if err != nil {
		return DoResult{
			RequestLabel: a.Name,
			Error:        err.Error(),
		}
	}
	return DoResult{RequestLabel: a.Name}
}

func (a *HTTPAttackerExample) Teardown() error {
	return nil
}
