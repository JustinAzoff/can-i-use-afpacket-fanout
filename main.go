package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"os/exec"

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
	workerID    int
)

func init() {
	flag.IntVar(&workerCount, "workercount", 8, "Number of workers")
	flag.IntVar(&fanoutGroup, "fanoutGroup", 42, "fanout group id")
	flag.IntVar(&maxFlows, "maxflows", 100, "How many flows to track before exiting")
	flag.StringVar(&iface, "interface", "eth0", "Interface")
	flag.IntVar(&statusInterval, "statusinterval", 500, "How many packets before each status update")
	flag.IntVar(&skipInitial, "skipinitial", 100, "How many packets to skip before collecting data")
	flag.IntVar(&workerID, "worker", 0, "worker id number (not for end users)")
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
	if tl == nil {
		return flow, fmt.Errorf("Nope")
	}
	if tl != nil {
		flow.proto = tl.LayerType().String()
		sport, dport := tl.TransportFlow().Endpoints()
		flow.sport = sport.String()
		flow.dport = dport.String()
	}
	return flow, nil
}

func spawn_worker(id int, flowchan chan WorkerFlow) {
	cmd := exec.Command(os.Args[0],
		"-skipinitial", fmt.Sprintf("%d", skipInitial),
		"-interface", iface,
		"-worker", fmt.Sprintf("%d", id),
		"-fanoutGroup", fmt.Sprintf("%d", fanoutGroup),
	)
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		log.Fatal(err)
	}
	if err := cmd.Start(); err != nil {
		log.Fatal(err)
	}

	for {
		var w WorkerFlow
		fmt.Fscanln(stdout, &w.workerID, &w.flow.proto, &w.flow.src, &w.flow.sport, &w.flow.dst, &w.flow.dport)
		flowchan <- w
	}

	if err := cmd.Wait(); err != nil {
		log.Fatal(err)
	}
}

func worker() {
	handle, err := afpacket.NewTPacket(afpacket.OptInterface(iface))
	if err != nil {
		log.Fatal(err)
	}
	err = handle.SetFanout(afpacket.FanoutHashWithDefrag, uint16(fanoutGroup))
	if err != nil {
		log.Fatal(err)
	}
	defer handle.Close()

	source := gopacket.NewPacketSource(handle, layers.LinkTypeEthernet)

	n := 0
	for packet := range source.Packets() {
		ft, err := getFiveTuple(packet)
		if err != nil {
			continue
		}
		if n > skipInitial {
			fmt.Println(workerID, ft.proto, ft.src, ft.sport, ft.dst, ft.dport)
		} else {
			n++
			if n == skipInitial {
				log.Printf("Worker %d has seen at least %d packets, collecting data", workerID, skipInitial)
			}
		}
	}
}

func main() {

	if workerID != 0 {
		worker()
		os.Exit(0)
	}

	flows := make(chan WorkerFlow, workerCount)

	flowMap := make(map[FiveTuple]int)
	failedFlowMap := make(map[FiveTuple]bool)
	successFlowMap := make(map[FiveTuple]bool)
	workerFlowCounts := make(map[int]int)

	for w := 1; w < workerCount+1; w++ {
		log.Printf("Starting worker id %d on interface %s", w, iface)
		go spawn_worker(w, flows)
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
		reverseFlow := FiveTuple{flow.proto, flow.dst, flow.dport, flow.src, flow.sport}

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
	for w := 1; w < workerCount+1; w++ {
		log.Printf(" - worker=%d flows=%d", w, workerFlowCounts[w])
	}
}
