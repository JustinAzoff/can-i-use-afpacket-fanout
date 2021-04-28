package main

import (
	"flag"
	"fmt"
	"log"
	"sync"

	"github.com/google/gopacket"
	"github.com/google/gopacket/afpacket"
	"github.com/google/gopacket/layers"
)

var (
	workerCount    int
	iface          string
	fanoutGroup    int
	maxFlows       int
	statusInterval int
	//skipInitial is used to skip packets that are delivered before the kernel fully sets up the load balancing
	//between all the workers
	skipInitial int
	includeNetworkLayer bool
	dumpNetworkLayerInformation bool
	wg          sync.WaitGroup
)

func init() {
	flag.IntVar(&workerCount, "workercount", 8, "Number of workers")
	flag.IntVar(&fanoutGroup, "fanoutGroup", 42, "fanout group id")
	flag.IntVar(&maxFlows, "maxflows", 100, "How many flows to track before exiting")
	flag.StringVar(&iface, "interface", "eth0", "Interface")
	flag.IntVar(&statusInterval, "statusinterval", 500, "How many packets before each status update")
	flag.IntVar(&skipInitial, "skipinitial", 100, "How many packets to skip before collecting data")
	flag.BoolVar(&includeNetworkLayer, "includenetworklayer", false, "Set this flag to include the link and network layer protocols in the hash calculation")
	flag.BoolVar(&dumpNetworkLayerInformation, "dumpnetworklayerinformation", false, "Set this flag to include the network layer information in the per-node output. Implies includenetworklayer")
	flag.Parse()
}

type FiveTuple struct {
	proto                  string
	src, sport, dst, dport string
	layerNames             string
}

type WorkerFlow struct {
	workerID int
	flow     FiveTuple
}

type Stats struct {
	packets         int
	success         int
	reverseSuccess  int
	failures        int
	reverseFailures int
}

func getFiveTuple(p gopacket.Packet) (FiveTuple, error) {
	var flow FiveTuple

	nl := p.NetworkLayer()
	if nl == nil {
		return flow, fmt.Errorf("Nope")
	}
	src, dst := nl.NetworkFlow().Endpoints()
	flow.src = src.String()
	flow.dst = dst.String()
	tl := p.TransportLayer()
	if tl != nil {
		flow.proto = tl.LayerType().String()
		sport, dport := tl.TransportFlow().Endpoints()
		flow.sport = sport.String()
		flow.dport = dport.String()
	}

	if includeNetworkLayer {
		for _, layer := range p.Layers() {
			// stop at the transport layer
			if tl != nil && layer == tl {
				break
			}
			if len(flow.layerNames) == 0 {
				flow.layerNames = layer.LayerType().String()
			} else {
				flow.layerNames += ", " + layer.LayerType().String()
			}
		}
	}

	return flow, nil
}

func worker(id int, flowchan chan WorkerFlow) {
	handle, err := afpacket.NewTPacket(afpacket.OptInterface(iface))
	if err != nil {
		log.Fatal(err)
	}
	err = handle.SetFanout(afpacket.FanoutHashWithDefrag, uint16(fanoutGroup))
	if err != nil {
		log.Fatal(err)
	}
	defer handle.Close()

	wg.Done()
	wg.Wait()

	source := gopacket.NewPacketSource(handle, layers.LinkTypeEthernet)

	n := 0
	for packet := range source.Packets() {
		ft, err := getFiveTuple(packet)
		if err != nil {
			continue
		}
		if n > skipInitial {
			flowchan <- WorkerFlow{id, ft}
		} else {
			n++
			if n == skipInitial {
				log.Printf("Worker %d has seen at least %d packets, collecting data", id, skipInitial)
			}
		}
	}
}

func main() {

	if dumpNetworkLayerInformation {
		includeNetworkLayer = true
	}

	flows := make(chan WorkerFlow, workerCount)

	flowMap := make(map[FiveTuple]int)
	failedFlowMap := make(map[FiveTuple]bool)
	successFlowMap := make(map[FiveTuple]bool)
	workerFlowCounts := make(map[int]int)
	workerProtocolCounts := make(map[int]map[string]int)

	wg.Add(workerCount)
	for w := 0; w < workerCount; w++ {
		log.Printf("Starting worker id %d on interface %s", w, iface)
		go worker(w, flows)
	}
	wg.Wait()
	log.Printf("%d workers started. Collecting results until %d flows have been seen..", workerCount, maxFlows)

	s := Stats{}
	for workerflow := range flows {
		s.packets++

		// Check if this flow was seen before, and if so, on the same worker
		flow := workerflow.flow
		worker, existed := flowMap[flow]
		if !existed {
			flowMap[flow] = workerflow.workerID
			workerFlowCounts[workerflow.workerID]++

			if dumpNetworkLayerInformation {
				// let's also do a protocol count
				protocols, ok := workerProtocolCounts[workerflow.workerID]
				if !ok {
					protocols = make(map[string]int)
					workerProtocolCounts[workerflow.workerID] = protocols
				}
				protocols[flow.layerNames]++
			}

		} else if worker != workerflow.workerID {
			log.Printf("FAIL: saw flow %s on worker %d expected %d", flow, workerflow.workerID, worker)
			failedFlowMap[flow] = true
			delete(successFlowMap, flow)
			s.failures++
		} else {
			if _, exists := failedFlowMap[flow]; !exists {
				successFlowMap[flow] = true
			}
			s.success++
		}


		//now check if the reverse flow was seen, and if so, on the same worker
		reverseFlow := FiveTuple{flow.proto, flow.dst, flow.dport, flow.src, flow.sport, flow.layerNames}

		worker, existed = flowMap[reverseFlow]
		if !existed {
			//Nothing to do in this case, can't draw any conclusions
		} else if worker != workerflow.workerID {
			log.Printf("FAIL: saw reverse flow of %s on worker %d expected %d", flow, workerflow.workerID, worker)
			failedFlowMap[reverseFlow] = true
			delete(successFlowMap, reverseFlow)
			s.reverseFailures++
		} else {
			if _, exists := failedFlowMap[reverseFlow]; !exists {
				successFlowMap[reverseFlow] = true
			}
			s.reverseSuccess++
		}
		if len(flowMap) > maxFlows {
			break
		}

		if s.packets%statusInterval == 0 {
			log.Printf("Stats: packets=%d flows=%d success_flows=%d failed_flows=%d pkt_success=%d pkt_reverse_success=%d pkt_failures=%d pkt_reverse_failures=%d",
				s.packets, len(flowMap), len(successFlowMap), len(failedFlowMap), s.success, s.reverseSuccess, s.failures, s.reverseFailures)
		}
	}
	log.Printf("Final Stats: packets=%d flows=%d success_flows=%d failed_flows=%d pkt_success=%d pkt_reverse_success=%d pkt_failures=%d pkt_reverse_failures=%d",
		s.packets, len(flowMap), len(successFlowMap), len(failedFlowMap), s.success, s.reverseSuccess, s.failures, s.reverseFailures)
	log.Printf("Worker flow count distribution:")
	for w := 0; w < workerCount; w++ {
		log.Printf(" - worker=%d flows=%d", w, workerFlowCounts[w])
		if dumpNetworkLayerInformation && workerFlowCounts[w] > 0 {
			for proto, count := range workerProtocolCounts[w] {
				log.Printf("   - protocol=%s flows=%d", proto, count)
			}
		}
	}
}
