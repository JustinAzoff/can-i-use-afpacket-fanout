package main

import (
	"flag"
	"fmt"
	"log"

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
)

func init() {
	flag.IntVar(&workerCount, "workercount", 8, "Number of workers")
	flag.IntVar(&fanoutGroup, "fanoutGroup", 42, "fanout group id")
	flag.IntVar(&maxFlows, "maxflows", 100, "How many flows to track before exiting")
	flag.StringVar(&iface, "interface", "eth0", "Interface")
	flag.IntVar(&statusInterval, "statusinterval", 500, "How many packets before each status update")
	flag.Parse()
}

type FiveTuple struct {
	proto                  string
	src, sport, dst, dport string
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
	return flow, nil
}

func worker(id int, flowchan chan WorkerFlow) {
	handle, err := afpacket.NewTPacket(afpacket.OptInterface(iface))
	if err != nil {
		log.Fatal(err)
	}
	err = handle.SetFanout(afpacket.FanoutHash, uint16(fanoutGroup))
	if err != nil {
		log.Fatal(err)
	}
	defer handle.Close()

	source := gopacket.NewPacketSource(handle, layers.LinkTypeEthernet)

	for packet := range source.Packets() {
		ft, err := getFiveTuple(packet)
		if err != nil {
			continue
		}
		flowchan <- WorkerFlow{id, ft}
	}
}

func main() {

	flows := make(chan WorkerFlow, workerCount)

	flowMap := make(map[FiveTuple]int)
	failedFlowMap := make(map[FiveTuple]bool)
	workerFlowCounts := make(map[int]int)

	for w := 0; w < workerCount; w++ {
		log.Printf("Starting worker id %d on interface %s", w, iface)
		go worker(w, flows)
	}
	log.Printf("Collecting results until %d flows have been seen..", maxFlows)

	s := Stats{}
	for workerflow := range flows {
		s.packets++

		//Check if this flow was seen before, and if so, on the same worker
		flow := workerflow.flow
		worker, existed := flowMap[flow]
		if !existed {
			flowMap[flow] = workerflow.workerID
			workerFlowCounts[workerflow.workerID]++
		} else if worker != workerflow.workerID {
			log.Printf("FAIL: saw flow %s on workers %d and %d", flow, workerflow.workerID, worker)
			failedFlowMap[flow] = true
			s.failures++
		} else {
			s.success++
		}

		//now check if the reverse flow was seen, and if so, on the same worker
		reverseFlow := FiveTuple{flow.proto, flow.dst, flow.dport, flow.src, flow.sport}

		worker, existed = flowMap[reverseFlow]
		if !existed {
			//Nothing to do in this case, can't draw any conclusions
		} else if worker != workerflow.workerID {
			log.Printf("FAIL: saw reverse flow of %s on workers %d and %d", flow, workerflow.workerID, worker)
			failedFlowMap[reverseFlow] = true
			s.reverseFailures++
		} else {
			s.reverseSuccess++
		}
		if len(flowMap) > maxFlows {
			break
		}

		if s.packets%statusInterval == 0 {
			log.Printf("Stats: packets=%d flows=%d failed_flows=%d success=%d reverse_success=%d failures=%d reverse_failures=%d",
				s.packets, len(flowMap), len(failedFlowMap), s.success, s.reverseSuccess, s.failures, s.reverseFailures)
		}
	}
	log.Printf("Final Stats: packets=%d flows=%d failed_flows=%d success=%d reverse_success=%d failures=%d reverse_failures=%d",
		s.packets, len(flowMap), len(failedFlowMap), s.success, s.reverseSuccess, s.failures, s.reverseFailures)
	log.Printf("Worker flow count distribution:")
	for w := 0; w < workerCount; w++ {
		log.Printf(" - worker=%d flows=%d", w, workerFlowCounts[w])
	}
}
