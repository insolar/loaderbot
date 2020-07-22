/*
 * // Copyright 2020 Insolar Network Ltd.
 * // All rights reserved.
 * // This material is licensed under the Insolar License version 1.0,
 * // available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.
 */

package loaderbot

import "time"

type AttackResult struct {
	// Step when request scheduled to execute
	nextMsg    struct{}
	begin, end time.Time
	elapsed    time.Duration
	doResult   DoResult
}

// DoResult is the return value of a Do call on an Attack.
type DoResult struct {
	// Label identifying the request that was send which is only used for reporting the Metrics.
	RequestLabel string
	// The error that happened when sending the request or receiving the response.
	Error error
	// The HTTP status code.
	StatusCode int
	// Number of bytes transferred when sending the request.
	BytesIn int64
	// Number of bytes transferred when receiving the response.
	BytesOut int64
}
