package loaderbot

import (
	"bytes"
	"context"
	"encoding/gob"
	"log"
	"net"
	"sync"
	"sync/atomic"

	jsoniter "github.com/json-iterator/go"
	"google.golang.org/grpc"
)

// TestPolicy only one runner is allowed per node
type TestPolicy struct {
	*sync.Mutex
	busy bool
}

func (m *TestPolicy) isBusy() bool {
	m.Lock()
	defer m.Unlock()
	return m.busy
}

func (m *TestPolicy) setBusy(b bool) {
	m.Lock()
	defer m.Unlock()
	m.busy = b
}

type server struct {
	policy           *TestPolicy
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
		if atomic.LoadInt64(&r.Failed) == 1 {
			r.L.Infof("runner failed, exiting")
			r.CancelFunc()
		}
	}
}

func (s *server) Status(_ context.Context, _ *StatusRequest) (*StatusResponse, error) {
	return &StatusResponse{
		Busy: s.policy.isBusy(),
	}, nil
}

// Run starts Runner and stream Results back to cluster client
func (s *server) Run(req *RunConfigRequest, srv Loader_RunServer) error {
	if s.policy.isBusy() {
		return nil
	}
	s.policy.setBusy(true)
	cfg := UnmarshalConfigGob(req.Config)

	r := NewRunner(&cfg, AttackerFromString(cfg.InstanceType), nil)
	cfgJson, _ := jsoniter.MarshalIndent(cfg, "", "    ")
	r.L.Infof("running task: %s", cfgJson)
	var ctx context.Context
	ctx, s.runnerCancelFunc = context.WithCancel(context.Background())
	go func() {
		_, _ = r.Run(ctx)
	}()
	r.streamResults(srv)
	s.policy.setBusy(false)
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
	RegisterLoaderServer(s, &server{
		policy: &TestPolicy{
			Mutex: &sync.Mutex{},
			busy:  false,
		},
	})
	log.Printf("running node on: %s", addr)
	go func() {
		if err := s.Serve(lis); err != nil {
			log.Fatalf("failed to serve: %v", err)
		}
	}()
	return s
}
