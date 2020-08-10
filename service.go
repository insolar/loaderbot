package loaderbot

import (
	"bytes"
	"context"
	"encoding/gob"
	"encoding/json"
	"log"
	"net"
	"sync/atomic"

	"google.golang.org/grpc"
)

type server struct {
	runnerCancelFunc context.CancelFunc
	UnimplementedLoaderServer
}

func (r *Runner) streamResults(srv Loader_RunServer) {
	for chunk := range r.OutResults {
		var b bytes.Buffer
		enc := gob.NewEncoder(&b)
		err := enc.Encode(chunk)
		if err != nil {
			r.L.Error(err)
		}
		if err := srv.Send(&ResultsResponse{ResultsChunk: b.Bytes()}); err != nil {
			r.L.Error(err)
		}
		// send last tick batch and shutdown, other nodes will be cancelled by client
		if r.Cfg.FailOnFirstError && atomic.LoadInt64(&r.Failed) == 1 {
			r.CancelFunc()
		}
	}
}

// Run starts Runner and stream Results back to cluster client
func (s *server) Run(req *RunConfigRequest, srv Loader_RunServer) error {
	cfg := UnmarshalConfigGob(req.Config)

	r := NewRunner(&cfg, &HTTPAttackerExample{}, nil)
	cfgJson, _ := json.MarshalIndent(cfg, "", "    ")
	r.L.Infof("running task: %s", cfgJson)
	var ctx context.Context
	ctx, s.runnerCancelFunc = context.WithCancel(context.Background())
	go func() {
		_, _ = r.Run(ctx)
	}()
	r.streamResults(srv)
	return nil
}

func (s *server) ShutdownNode(_ context.Context, _ *ShutdownNodeRequest) (*ShutdownNodeResponse, error) {
	if s.runnerCancelFunc != nil {
		s.runnerCancelFunc()
	}
	return &ShutdownNodeResponse{}, nil
}

func RunService(addr string) *grpc.Server {
	lis, err := net.Listen("tcp", addr)
	if err != nil {
		log.Fatal(err)
	}
	s := grpc.NewServer()
	RegisterLoaderServer(s, &server{})
	log.Printf("running node on: %s", addr)
	go func() {
		if err := s.Serve(lis); err != nil {
			log.Fatalf("failed to serve: %v", err)
		}
	}()
	return s
}
