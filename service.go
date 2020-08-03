package loaderbot

import (
	"bytes"
	"encoding/gob"
	"encoding/json"
	"google.golang.org/grpc"
	"log"
	"net"
)

type server struct {
	UnimplementedLoaderServer
}

// for ease of use cfg now is just bytes, create pb types later
func MarshalConfigGob(cfg interface{}) []byte {
	var b bytes.Buffer
	enc := gob.NewEncoder(&b)
	if err := enc.Encode(cfg); err != nil {
		log.Fatal(err)
	}
	return b.Bytes()
}

func UnmarshalConfigGob(d []byte) RunnerConfig {
	b := bytes.NewBuffer(d)
	dec := gob.NewDecoder(b)
	var cfg RunnerConfig
	if err := dec.Decode(&cfg); err != nil {
		log.Fatal(err)
	}
	return cfg
}

func streamResults(r *Runner, srv Loader_RunServer) {
	for chunk := range r.OutResults {
		var b bytes.Buffer
		enc := gob.NewEncoder(&b)
		err := enc.Encode(chunk)
		if err != nil {
			log.Fatal(err)
		}
		if err := srv.Send(&ResultsResponse{ResultsChunk: b.Bytes()}); err != nil {
			log.Fatal(err)
		}
	}
}

// Run starts Runner and stream results back to cluster client
func (s *server) Run(req *RunConfigRequest, srv Loader_RunServer) error {
	cfg := UnmarshalConfigGob(req.Config)

	r := NewRunner(&cfg, &HTTPAttackerExample{}, nil)
	cfgJson, _ := json.Marshal(cfg)
	r.L.Infof("running task: %s", cfgJson)
	go r.Run()
	streamResults(r, srv)
	return nil
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
