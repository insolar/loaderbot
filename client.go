package loaderbot

import (
	"bytes"
	"context"
	"encoding/gob"
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

func (m *NodeClient) StartRunner(cluster *ClusterClient) {
	var err error
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()
	m.stream, err = m.Run(ctx, &RunConfigRequest{
		Config:       MarshalConfigGob(cluster.testCfg),
		AttackerName: "http",
	})
	if err != nil {
		cluster.L.Fatal(err)
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
			cluster.L.Fatal(err)
		}
		if err != nil {
			cluster.L.Error(err)
		}
		cluster.Results <- results
	}
}

type ClusterClient struct {
	TimeoutCtx context.Context
	CancelFunc context.CancelFunc

	failed             bool
	testCfg            *RunnerConfig
	activeClients      int32
	clients            []*NodeClient
	Results            chan []AttackResult
	clusterTickMetrics map[int]*ClusterTickMetrics
	L                  *Logger
}

func NewClusterClient(cfg *RunnerConfig) *ClusterClient {
	clients := make([]*NodeClient, 0)
	for _, addr := range cfg.ClusterOptions.Nodes {
		clients = append(clients, NewNodeClient(addr))
	}
	return &ClusterClient{
		testCfg:            cfg,
		clients:            clients,
		Results:            make(chan []AttackResult),
		clusterTickMetrics: make(map[int]*ClusterTickMetrics),
		L:                  NewLogger(cfg).With("cluster", cfg.Name),
	}
}

func (m *ClusterClient) Run() {
	for _, c := range m.clients {
		atomic.AddInt32(&m.activeClients, 1)
		go c.StartRunner(m)
	}
	m.collectResults()
	for _, c := range m.clients {
		c.Close()
	}
}

func (m *ClusterClient) collectResults() {
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
			if len(currentTickMetrics.Samples) == len(m.testCfg.ClusterOptions.Nodes) {
				for _, sampleBatch := range currentTickMetrics.Samples {
					for _, s := range sampleBatch {
						currentTickMetrics.Metrics.add(s)
					}
				}
				currentTickMetrics.Metrics.update()
				m.L.Infof(
					"step: %d, tick: %d, rate [%4f -> %v], perc: 50 [%v] 95 [%v] 99 [%v], # requests [%d], %% success [%d]",
					token.Step,
					tick,
					// TODO: fix this to fired RPS aggregation
					currentTickMetrics.Metrics.Rate,
					token.TargetRPS*len(m.testCfg.ClusterOptions.Nodes),
					currentTickMetrics.Metrics.Latencies.P50,
					currentTickMetrics.Metrics.Latencies.P95,
					currentTickMetrics.Metrics.Latencies.P99,
					currentTickMetrics.Metrics.Requests,
					currentTickMetrics.Metrics.successLogEntry(),
				)
				// shutdown other Nodes if error is present in tick
				if m.shutdownOnNodeSampleError(currentTickMetrics) {
					return
				}
			}
		default:
			if atomic.LoadInt32(&m.activeClients) == 0 {
				m.L.Infof("all nodes exited, test ended")
				return
			}
		}
	}
}

func (m *ClusterClient) shutdownOnNodeSampleError(clusterTick *ClusterTickMetrics) bool {
	for _, sampleBatch := range clusterTick.Samples {
		for _, s := range sampleBatch {
			if s.DoResult.Error != "" && m.testCfg.FailOnFirstError {
				for idx, c := range m.clients {
					m.L.Infof("shutting down runner: %d", idx)
					c.Shutdown()
				}
				m.failed = true
				return true
			}
		}
	}
	return false
}
