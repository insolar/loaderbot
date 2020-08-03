package loaderbot

import (
	"bytes"
	"context"
	"encoding/gob"
	"fmt"
	"io"
	"log"
	"sync/atomic"
	"time"

	"google.golang.org/grpc"
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

func (m *NodeClient) StartRunner(cfg RunnerConfig, cluster *ClusterClient) {
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
	m.receive(cluster)
}

func (m *NodeClient) Shutdown() {
	if _, err := m.LoaderClient.ShutdownNode(context.Background(), &ShutdownNodeRequest{}); err != nil {
		log.Fatal(err)
	}
}

func (m *NodeClient) Close() {
	m.conn.Close()
}

func (m *NodeClient) receive(cluster *ClusterClient) {
	for {
		res, err := m.stream.Recv()
		if res == nil {
			atomic.AddInt32(&cluster.activeClients, -1)
			return
		}
		if err == io.EOF {
			atomic.AddInt32(&cluster.activeClients, -1)
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
		cluster.Results <- results
	}
}

type ClusterClient struct {
	totalClients       int
	activeClients      int32
	clients            []*NodeClient
	Results            chan []AttackResult
	clusterTickMetrics map[int]*ClusterTickMetrics
}

func NewClusterClient(totalClients int) *ClusterClient {
	clients := make([]*NodeClient, 0)
	port := 50051
	for i := 0; i < totalClients; i++ {
		clients = append(clients, NewNodeClient(fmt.Sprintf("localhost:%d", port+1)))
	}
	return &ClusterClient{
		totalClients:       totalClients,
		clients:            clients,
		Results:            make(chan []AttackResult),
		clusterTickMetrics: make(map[int]*ClusterTickMetrics),
	}
}

func (m *ClusterClient) RunClusterTestFromConfig(cfg RunnerConfig) {
	for _, c := range m.clients {
		atomic.AddInt32(&m.activeClients, 1)
		go c.StartRunner(cfg, m)
	}
	m.collectResults(cfg)
	for _, c := range m.clients {
		c.Close()
	}
}

func (m *ClusterClient) collectResults(cfg RunnerConfig) {
	for {
		select {
		case res := <-m.Results:
			token := res[0].AttackToken
			tick := res[0].AttackToken.Tick
			if _, ok := m.clusterTickMetrics[tick]; !ok {
				m.clusterTickMetrics[tick] = &ClusterTickMetrics{
					Samples: make([][]AttackResult, 0),
					Metrics: NewMetrics(),
				}
			}
			currentTickMetrics := m.clusterTickMetrics[tick]
			currentTickMetrics.Samples = append(currentTickMetrics.Samples, res)
			if len(currentTickMetrics.Samples) == m.totalClients {
				for _, sampleBatch := range currentTickMetrics.Samples {
					for _, s := range sampleBatch {
						currentTickMetrics.Metrics.add(s)
					}
				}
				currentTickMetrics.Metrics.update()
				log.Printf(
					"CLUSTER step: %d, tick: %d, rate [%4f -> %v], perc: 50 [%v] 95 [%v] 99 [%v], # requests [%d], %% success [%d]",
					token.Step,
					tick,
					currentTickMetrics.Metrics.Rate,
					token.TargetRPS*m.totalClients,
					currentTickMetrics.Metrics.Latencies.P50,
					currentTickMetrics.Metrics.Latencies.P95,
					currentTickMetrics.Metrics.Latencies.P99,
					currentTickMetrics.Metrics.Requests,
					currentTickMetrics.Metrics.successLogEntry(),
				)
			}
			// shutdown other clients if error
			for _, sampleBatch := range currentTickMetrics.Samples {
				for _, s := range sampleBatch {
					if s.DoResult.Error != nil && cfg.FailOnFirstError {
						for _, c := range m.clients {
							c.Shutdown()
						}
					}
				}
			}
		default:
			if atomic.LoadInt32(&m.activeClients) == 0 {
				log.Printf("all clients exited, test ended")
				return
			}
		}
	}
}
