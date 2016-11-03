package main

import (
	"flag"
	"fmt"
	"log"
	"time"

	"github.com/google/gopacket"
	"github.com/google/gopacket/afpacket"
	"github.com/google/gopacket/layers"
)

var (
	workerCount int
	iface       string
	fanoutGroup int
	maxFlows    int
)

func init() {
	flag.IntVar(&workerCount, "workercount", 8, "Number of workers")
	flag.IntVar(&fanoutGroup, "fanoutGroup", 42, "fanout group id")
	flag.IntVar(&maxFlows, "maxflows", 100, "How many flows to track before exiting")
	flag.StringVar(&iface, "interface", "eth0", "Interface")
	flag.Parse()
}

type FiveTuple struct {
	proto                  string
	src, sport, dst, dport string
}

type WorkerFlow struct {
	workerId int
	flow     FiveTuple
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
	log.Printf("Starting worker id %d on interface %s", id, iface)
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

	for w := 0; w < workerCount; w++ {
		go worker(w, flows)
		time.Sleep(10 * time.Millisecond)
	}

	packets := 0
	success := 0
	failures := 0
	reverseFailures := 0
	for workerflow := range flows {
		packets++

		//Check if this flow was seen before, and if so, on the same worker
		flow := workerflow.flow
		worker, existed := flowMap[flow]
		if !existed {
			flowMap[flow] = workerflow.workerId
		} else if worker != workerflow.workerId {
			log.Printf("FAIL: saw flow %s on workers %d and %d", flow, workerflow.workerId, worker)
			failures++
		} else {
			success++
		}

		//now check if the reverse flow was seen, and if so, on the same worker
		reverseFlow := FiveTuple{flow.proto, flow.dst, flow.dport, flow.src, flow.sport}

		worker, existed = flowMap[reverseFlow]
		if !existed {
			//Nothing to do in this case
		} else if worker != workerflow.workerId {
			log.Printf("FAIL: saw reverse flow of %s on workers %d and %d", flow, workerflow.workerId, worker)
			reverseFailures++
		} else {
			success++
		}

		if packets%100 == 0 {
			log.Printf("packets=%d flows=%d success=%d failures=%d reverse_failures=%d", packets, len(flowMap), success, failures, reverseFailures)
		}
		if len(flowMap) > maxFlows {
			break
		}
	}
	log.Printf("packets=%d flows=%d success=%d failures=%d reverse_failures=%d", packets, len(flowMap), success, failures, reverseFailures)
}
