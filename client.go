package loaderbot

import (
	"bytes"
	"context"
	"encoding/gob"
	"fmt"
	"google.golang.org/grpc"
	"io"
	"log"
	"time"
)

type NodeClient struct {
	conn *grpc.ClientConn
	LoaderClient
	stream Loader_RunClient
}

func NewNodeClient(addr string) *NodeClient {
	conn, err := grpc.Dial(addr, grpc.WithInsecure(), grpc.WithBlock())
	if err != nil {
		log.Fatalf("failed to connect node %s: %v", addr, err)
	}
	return &NodeClient{
		conn:         conn,
		LoaderClient: NewLoaderClient(conn),
		stream:       nil,
	}
}

func (m *NodeClient) StartRunner(cfg RunnerConfig, aggregateResults chan []AttackResult) {
	var err error
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()
	m.stream, err = m.Run(ctx, &RunConfigRequest{
		Config:       MarshalConfigGob(cfg),
		AttackerName: "http",
	})
	if err != nil {
		log.Fatal(err)
	}
	m.receive(aggregateResults)
}

func (m *NodeClient) Close() {
	m.conn.Close()
}

func (m *NodeClient) receive(aggregateResults chan []AttackResult) {
	for {
		res, err := m.stream.Recv()
		if res == nil {
			return
		}
		if err == io.EOF {
			return
		}
		b := bytes.NewBuffer(res.ResultsChunk)
		dec := gob.NewDecoder(b)
		var results []AttackResult
		if err := dec.Decode(&results); err != nil {
			log.Fatal(err)
		}
		if err != nil {
			log.Fatal(err)
		}
		//log.Printf("Data: %s", results)
		aggregateResults <- results
	}
}

type ClusterClient struct {
	clients            []*NodeClient
	results            chan []AttackResult
	clusterTickMetrics map[int]*TickMetrics
}

func NewClusterClient() *ClusterClient {
	clients := make([]*NodeClient, 0)
	port := 50051
	for i := 0; i < 2; i++ {
		clients = append(clients, NewNodeClient(fmt.Sprintf("localhost:%d", port+1)))
	}
	return &ClusterClient{
		clients:            clients,
		results:            make(chan []AttackResult),
		clusterTickMetrics: make(map[int]*TickMetrics),
	}
}

func (m *ClusterClient) RunClusterTestFromConfig(cfg RunnerConfig) {
	for _, c := range m.clients {
		go c.StartRunner(cfg, m.results)
	}
	m.collectResults()
	for _, c := range m.clients {
		c.Close()
	}
}

func (m *ClusterClient) collectResults() {
	for {
		select {
		case res := <-m.results:
			log.Printf("res: %s", res)
		}
	}
}
