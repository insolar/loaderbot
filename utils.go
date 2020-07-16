package loaderbot

import (
	"os"
	"os/signal"
	"runtime"
	"syscall"
)

var (
	sigs = make(chan os.Signal, 1)
)

func (r *Runner) handleShutdownSignal() {
	signal.Notify(sigs, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		<-sigs
		r.L.Infof("exit signal received, exiting")
		buf := make([]byte, 1<<20)
		stacklen := runtime.Stack(buf, true)
		r.L.Infof("=== received SIGTERM ===\n*** goroutine dump...\n%s\n*** end\n", buf[:stacklen])
		os.Exit(1)
	}()
}
