/*
 * // Copyright 2020 Insolar Network Ltd.
 * // All rights reserved.
 * // This material is licensed under the Insolar License version 1.0,
 * // available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.
 */

package loaderbot

import (
	"fmt"
	"time"
)

type AttackResult struct {
	AttackToken attackToken
	Begin, End  time.Time
	Elapsed     time.Duration
	DoResult    DoResult
}

func (a AttackResult) String() string {
	return fmt.Sprintf(
		"Begin: %s, End: %s, Elapsed: %d, token: [%s], doResult: %v",
		a.Begin.Format(time.RFC3339),
		a.End.Format(time.RFC3339),
		a.Elapsed,
		a.AttackToken,
		a.DoResult,
	)
}

// DoResult is the return value of a Do call on an Attack.
type DoResult struct {
	// Label identifying the request that was send which is only used for reporting the Metrics.
	RequestLabel string
	// The error that happened when sending the request or receiving the response.
	Error string
	// The HTTP status code.
	StatusCode int
	// Number of bytes transferred when sending the request.
	BytesIn int64
	// Number of bytes transferred when receiving the response.
	BytesOut int64
}
