package loaderbot

import (
	"fmt"
	"io"
	"log"
	"net/http"
	"net/http/pprof"
	"os"
	"runtime/trace"
	"time"

	"github.com/google/uuid"
)

// nolint
func pprofHandlers(r *http.ServeMux) {
	r.HandleFunc("/debug/pprof/", pprof.Index)
	r.HandleFunc("/debug/pprof/cmdline", pprof.Cmdline)
	r.HandleFunc("/debug/pprof/profile", pprof.Profile)
	r.HandleFunc("/debug/pprof/symbol", pprof.Symbol)
	r.HandleFunc("/debug/pprof/trace", pprof.Trace)
}

// nolint
func startTrace(f io.Writer) {
	go func() {
		_ = trace.Start(f)
	}()
}

// nolint
func pprofTrace(prefix string, secs int) {
	go func() {
		m := http.NewServeMux()
		pprofHandlers(m)
		if err := http.ListenAndServe(":8081", m); err != nil {
			log.Fatal(err)
		}
	}()
	f, err := os.Create(fmt.Sprintf("trace-%s-%s.out", prefix, uuid.New().String()))
	if err != nil {
		log.Fatal(err)
	}
	startTrace(f)
	time.Sleep(time.Duration(secs) * time.Second)
	trace.Stop()
}
