/*
 * // Copyright 2020 Insolar Network Ltd.
 * // All rights reserved.
 * // This material is licensed under the Insolar License version 1.0,
 * // available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.
 */

package main

import (
	"time"

	"github.com/insolar/loaderbot"
)

func main() {
	loaderbot.RunTestServer("0.0.0.0:9031", 1*time.Nanosecond)
	select {}
}
