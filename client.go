package loaderbot

import (
	"bytes"
	"context"
	"encoding/gob"
	"io"
	"log"
	"sync/atomic"
	"time"

	"github.com/jinzhu/copier"
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
	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(cluster.testCfg.TestTimeSec)*time.Second)
	defer cancel()
	nodeCfg := cluster.configToNodes()
	m.stream, err = m.Run(ctx, &RunConfigRequest{
		Config:       MarshalConfigGob(nodeCfg),
		AttackerName: "http",
	})
	if err != nil {
		cluster.L.Fatal(err)
	}
	m.receive(ctx, cluster)
}

func (m *NodeClient) Shutdown() {
	if _, err := m.LoaderClient.ShutdownNode(context.Background(), &ShutdownNodeRequest{}); err != nil {
		log.Fatal(err)
	}
}

func (m *NodeClient) Close() {
	m.conn.Close()
}

func (m *NodeClient) receive(_ context.Context, cluster *ClusterClient) {
	for {
		res, err := m.stream.Recv()
		if err == io.EOF || res == nil {
			atomic.AddInt32(&cluster.activeClients, -1)
			return
		}
		if err != nil {
			atomic.AddInt32(&cluster.activeClients, -1)
			cluster.L.Error(err)
		}
		b := bytes.NewBuffer(res.ResultsChunk)
		dec := gob.NewDecoder(b)
		var results []AttackResult
		if err := dec.Decode(&results); err != nil {
			cluster.L.Fatal(err)
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
	Report             *Report
	L                  *Logger
}

func NewClusterClient(cfg *RunnerConfig) *ClusterClient {
	clients := make([]*NodeClient, 0)
	var failed bool
	for _, addr := range cfg.ClusterOptions.Nodes {
		c := NewNodeClient(addr)
		res, err := c.Status(context.Background(), &StatusRequest{})
		if err != nil {
			log.Fatal(err)
		}
		if res.Busy {
			log.Printf(errNodeIsBusy, addr)
			failed = true
		}
		clients = append(clients, c)
	}
	c := &ClusterClient{
		testCfg:            cfg,
		clients:            clients,
		Results:            make(chan []AttackResult),
		clusterTickMetrics: make(map[int]*ClusterTickMetrics),
		failed:             failed,
		L:                  NewLogger(cfg).With("cluster", cfg.Name),
	}
	if cfg.ReportOptions.CSV {
		c.Report = NewReport(cfg)
	}
	return c
}

func (m *ClusterClient) configToNodes() *RunnerConfig {
	var nodeTestCfg RunnerConfig
	if err := copier.Copy(&nodeTestCfg, &m.testCfg); err != nil {
		m.L.Fatal(err)
	}
	// no need to write logs on nodes in cluster mode
	nodeTestCfg.ReportOptions.CSV = false
	nodeTestCfg.ReportOptions.PNG = false
	// split start/step rps by nodes equally
	nodeTestCfg.Attackers = nodeTestCfg.Attackers / len(nodeTestCfg.ClusterOptions.Nodes)
	nodeTestCfg.StartRPS = nodeTestCfg.StartRPS / len(nodeTestCfg.ClusterOptions.Nodes)
	nodeTestCfg.StepRPS = nodeTestCfg.StepRPS / len(nodeTestCfg.ClusterOptions.Nodes)
	return &nodeTestCfg
}

func (m *ClusterClient) CheckBusy() bool {
	res, err := m.clients[0].Status(context.Background(), &StatusRequest{})
	if err != nil {
		m.L.Error(err)
	}
	return res.Busy
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
	if m.testCfg.ReportOptions.CSV {
		m.Report.flushLogs()
		m.Report.plot()
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
				// aggregate over all ticks across cluster
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
					currentTickMetrics.Metrics.Rate,
					token.TargetRPS*len(m.testCfg.ClusterOptions.Nodes),
					currentTickMetrics.Metrics.Latencies.P50,
					currentTickMetrics.Metrics.Latencies.P95,
					currentTickMetrics.Metrics.Latencies.P99,
					currentTickMetrics.Metrics.Requests,
					currentTickMetrics.Metrics.successLogEntry(),
				)
				if m.testCfg.ReportOptions.CSV {
					m.Report.writePercentilesEntry(res[0], currentTickMetrics.Metrics)
				}
				// shutdown other Nodes if error is present in tick
				if m.shutdownOnNodeSampleError(currentTickMetrics.Metrics) {
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

func (m *ClusterClient) shutdownOnNodeSampleError(metrics *Metrics) bool {
	if metrics.Success < m.testCfg.SuccessRatio {
		for idx, c := range m.clients {
			m.L.Infof("shutting down runner: %d", idx)
			c.Shutdown()
		}
		m.failed = true
		return true
	}
	return false
}
