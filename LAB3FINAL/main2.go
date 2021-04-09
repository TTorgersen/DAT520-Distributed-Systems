//package main

import (
	fd "dat520/lab3/failuredetector"
	ld "dat520/lab3/leaderdetector"
	mp "dat520/lab4/multipaxos"
	network "dat520/test/network"
	"flag"
	"fmt"
	"net"
	"os"
	"strconv"
	"time"
)

var (
	help = flag.Bool(
		"help",
		false,
		"Show usage help",
	)
	ports = flag.Int(
		"ports",
		20043,
		"Ports for all the servers",
	)
	endpoint = flag.String(
		"endpoint",
		"pitter14.ux.uis.no",
		"Endpoint for this server, only need the IP or machine, port not needed",
	)
	id = flag.Int(
		"id",
		0,
		"Id of this process",
	)
	delay = flag.Int(
		"delay",
		1000,
		"Delay used by Increasing Timout failuredetector in milliseconds",
	)
)

/*
type Node struct {
	ID        int
	IP        string
	Port      int
	TCPaddr   *net.TCPAddr
	TCPListen *net.TCPListener
}
*/

func usage() {
	fmt.Fprintf(os.Stderr, "Usage: %s [OPTIONS]\n", os.Args[0])
	fmt.Fprintf(os.Stderr, "\nOptions:\n")
	flag.PrintDefaults()
}

//var Mutex = &sync.Mutex{}

func main() {
	flag.Usage = usage
	flag.Parse()
	if *help {
		flag.Usage()
		os.Exit(0)
	}
	hardcoded := [3]string{
		"pitter14.ux.uis.no" + ":" + strconv.Itoa(*ports),
		"pitter16.ux.uis.no" + ":" + strconv.Itoa(*ports),
		"pitter3.ux.uis.no" + ":" + strconv.Itoa(*ports),
	}
	nodeIDs := []int{0, 1, 2}
	// Set up self node
	selfAddress, err := net.ResolveTCPAddr("tcp", *endpoint+":"+strconv.Itoa(*ports))
	check(err)
	selfNode := network.Node{
		ID:      *id,
		IP:      selfAddress.IP.String(),
		Port:    selfAddress.Port,
		TCPaddr: selfAddress,
	}
	// Make list of other nodes in network
	nodes := []network.Node{}
	for i, addr := range hardcoded {
		a, err := net.ResolveTCPAddr("tcp", addr)
		check(err)
		if i != selfNode.ID {
			nodes = append(nodes, network.Node{
				ID:      i,
				IP:      a.IP.String(),
				Port:    a.Port,
				TCPaddr: a,
			})
		}
	}
	// Initialize the network
	thisNetwork := network.InitializeNetwork(nodes, selfNode)

	// Set up leader and failuredetector
	hbSend := make(chan fd.Heartbeat)
	leaderdetector := ld.NewMonLeaderDetector(nodeIDs)
	failuredetector := fd.NewEvtFailureDetector(*id, nodeIDs, leaderdetector, time.Duration(*delay)*time.Millisecond, hbSend)
	failuredetector.Start()

	// Set up multipaxos
	//INIT MULTIPAXOS proposer
	prepareOut := make(chan mp.Prepare, 2000000) //sendonly channel send prepare
	acceptOut := make(chan mp.Accept, 2000000)   //Incoming accept
	//adu -1 is initially
	proposer := mp.NewProposer(thisNetwork.Myself.ID, len(nodeIDs), -1, leaderdetector, prepareOut, acceptOut)

	//Step 4.6 INIT acceptor
	promiseOut := make(chan mp.Promise, 2000000) //send promises to other nodes
	learnOut := make(chan mp.Learn, 2000000)     //send learn to other nodes
	acceptor := mp.NewAcceptor(thisNetwork.Myself.ID, promiseOut, learnOut)
	//step 4.7 INIT learner
	decidedOut := make(chan mp.DecidedValue, 2000000) //send values that has been learned
	learner := mp.NewLearner(thisNetwork.Myself.ID, len(nodeIDs), decidedOut)

	// start up the network
	thisNetwork.Dial()
	thisNetwork.StartServer()

	// Start multipaxos
	proposer.Start()
	acceptor.Start()
	learner.Start()

	fmt.Println("Starting server: ", selfAddress, " With id: ", *id)

	go subscribePrinter(leaderdetector.Subscribe())
	for {
		select {
		// make heartbeat message and send to receiver
		case hb := <-hbSend:
			hbMsg := network.Message{
				Type:      "Heartbeat",
				From:      hb.From,
				To:        hb.To,
				Heartbeat: hb,
			}
			thisNetwork.SendMessage(hbMsg)
		// make prepare message and send to network
		case prp := <-prepareOut:
			fmt.Println("Sends prepare out to network")
			prpMsg := network.Message{
				Type:    "Prepare",
				From:    prp.From,
				Prepare: prp,
			}
			thisNetwork.SendMessageBroadcast(prpMsg, nodeIDs)
		// make accept message and send to network
		case acc := <-acceptOut:
			fmt.Println("Sends accept out to network")
			accMsg := network.Message{
				Type:   "Accept",
				From:   acc.From,
				Accept: acc,
			}
			thisNetwork.SendMessageBroadcast(accMsg, nodeIDs)
		// make promise message and send to receiver
		case prm := <-promiseOut:
			fmt.Println("Sends promise out to network")
			prmMsg := network.Message{
				Type:    "Promise",
				To:      prm.To,
				From:    prm.From,
				Promise: prm,
			}
			thisNetwork.SendMessage(prmMsg)
		// make learn message and send to network
		case lrn := <-learnOut:
			fmt.Println("Sends learn out to network")
			lrnMsg := network.Message{
				Type:  "Learn",
				From:  lrn.From,
				Learn: lrn,
			}
			thisNetwork.SendMessageBroadcast(lrnMsg, nodeIDs)
		// make response message and send to client
		case response := <-decidedOut:
			resp := mp.Response{
				ClientID:  response.Value.ClientID,
				ClientSeq: response.Value.ClientSeq,
				Command:   response.Value.Command,
			}
			resMsg := network.Message{
				Type:     "Response",
				Response: resp,
				ClientIP: response.Value.ClientID,
			}
			fmt.Println("Sends response to client")
			thisNetwork.SendMessage(resMsg)
			proposer.IncrementAllDecidedUpTo()
		// message on network to on self
		case msg := <-thisNetwork.RecieveChannel:
			switch {
			case msg.Type == "Heartbeat":
				failuredetector.DeliverHeartbeat(msg.Heartbeat)
			case msg.Type == "Prepare":
				acceptor.DeliverPrepare(msg.Prepare)
			case msg.Type == "Promise":
				fmt.Println("Deliver promise to proposer")
				proposer.DeliverPromise(msg.Promise)
			case msg.Type == "Accept":
				fmt.Println("Deliver accept to acceptor")
				acceptor.DeliverAccept(msg.Accept)
			case msg.Type == "Learn":
				fmt.Println("Deliver learn to learner")
				learner.DeliverLearn(msg.Learn)
			case msg.Type == "Value":
				fmt.Println("Deliver value from client to proposer")
				proposer.DeliverClientValue(msg.Value)
			case msg.Type == "Responce":
				fmt.Println(msg.Response)
			}
		}
	}
}
func subscribePrinter(sub <-chan int) {
	for {
		fmt.Print(<-sub)
		fmt.Println(" Leader change")
	}
}

func check(err error) {
	if err != nil {
		fmt.Fprintf(os.Stderr, "Fatal error: %s\n", err.Error())
		os.Exit(1)
	}
}