package main

import (
	"time"

	"github.com/insolar/loaderbot"
)

func main() {
	loaderbot.RunTestServer("0.0.0.0:9031", 1*time.Nanosecond)
	select {}
}
