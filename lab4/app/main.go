package main

import (
	detector "dat520/lab3/detector"
	mp "dat520/lab4/multipaxos"
	"dat520/lab4/network"
	"encoding/json"
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"os"
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

func usage() {
	fmt.Fprintf(os.Stderr, "Usage: %s [OPTIONS]\n", os.Args[0])
	fmt.Fprintf(os.Stderr, "\nOptions:\n")
	flag.PrintDefaults()
}

func main() {
	fmt.Println("Main application")
	flag.Usage = usage
	flag.Parse()
	if *help {
		flag.Usage()
		os.Exit(0)
	}
	// step 1: get network configuration

	//1.1 get network configuration file
	netconfigureFile, err := os.Open("netconf.json")
	check(err)
	defer netconfigureFile.Close()

	//1.2 Read the network config file as byte array
	byteVal, _ := ioutil.ReadAll(netconfigureFile)

	//initialize a netconfig struct to save the netconfig
	var netconf network.Netconf
	err = json.Unmarshal(byteVal, &netconf)
	check(err)

	//1.3 Now netconf has the info from json file, step 1 complete
	fmt.Println(netconf)

	// step 2: initialize a empty network with channels
	thisNetwork, err := network.InitializeNetwork(netconf.Nodes, *id)
	check(err)
	fmt.Println(thisNetwork)
	// step 2.1: now we have a network with tcp endpoints, all nodes present

	// step 3: Initialize LD

	// step 3.1: in order to Init Leaderdetector we need a list of all nodes present
	nodeIDList := []int{}
	for _, node := range thisNetwork.Nodes {
		nodeIDList = append(nodeIDList, node.ID)
	}

	//as we dont have myself in the nodelist we need to append thatone too
	nodeIDList = append(nodeIDList, thisNetwork.Myself.ID)

	ld := detector.NewMonLeaderDetector(nodeIDList)

	// step 4: initialize FD

	//step 4.1 as failiuredetector requires a hb channel  we must create one
	hbSend := make(chan detector.Heartbeat, 16)
	fd := detector.NewEvtFailureDetector(thisNetwork.Myself.ID, nodeIDList, ld, 10*time.Second, hbSend)

	// step 3 and 4 complete, now we have a failiure detector
	fmt.Println(fd)

	//step 4.5 : INIT MULTIPAXOS proposer
	prepareOut := make(chan mp.Prepare, 2000000) //sendonly channel send prepare
	acceptOut := make(chan mp.Accept, 2000000)   //Incoming accept
	//adu -1 is initially
	proposer := mp.NewProposer(thisNetwork.Myself.ID, len(nodeIDList), -1, ld, prepareOut, acceptOut)

	//Step 4.6 INIT acceptor
	promiseOut := make(chan mp.Promise, 2000000) //send promises to other nodes
	learnOut := make(chan mp.Learn, 2000000)     //send learn to other nodes
	acceptor := mp.NewAcceptor(thisNetwork.Myself.ID, promiseOut, learnOut)
	//step 4.7 INIT learner
	decidedOut := make(chan mp.DecidedValue, 2000000) //send values that has been learned
	learner := mp.NewLearner(thisNetwork.Myself.ID, len(nodeIDList), decidedOut)

	// step 5: Initialize connections
	// step 6: start server
	fmt.Println("Trying to init connections")
	thisNetwork.InitializeConnections()
	thisNetwork.StartServer()
	fmt.Println("Started server, we good")

	//start multiPax
	proposer.Start()
	acceptor.Start()
	learner.Start()

	ldchange := ld.Subscribe()
	fmt.Println("Leader is", ld.Leader())
	fd.Start()
	for {

		select {
		case newLeader := <-ldchange:
			fmt.Println("A NEW LEADER HAS BEEN CHOSEN, ALL HAIL LEADER ", newLeader)
			//fmt.Printf("\nSuspected nodes at ld: %v\n", ld.Suspected)
		case hb := <-hbSend:
			//fmt.Println("sends hb from", hb.From, " to  ", hb.To, " from hbsend to network send channel")
			sendHBMessage := network.Message{
				To:      hb.To,
				Type:    "Heartbeat",
				From:    hb.From,
				Request: hb.Request,
			}
			thisNetwork.SendChannel <- sendHBMessage
		case prp := <-prepareOut:
			fmt.Println("Sends prepare out to network")
			prpMsg := network.Message{
				Type:    "Prepare",
				From:    prp.From,
				Prepare: prp,
			}
			thisNetwork.SendMessageBroadcast(prpMsg, []int{0, 1, 2})
		case acc := <-acceptOut:
			fmt.Println("Sends accept out to network")
			accMsg := network.Message{
				Type:   "Accept",
				From:   acc.From,
				Accept: acc,
			}
			thisNetwork.SendMessageBroadcast(accMsg, []int{0, 1, 2})
		case prm := <-promiseOut:
			fmt.Println("Sends promise out to network")
			prmMsg := network.Message{
				Type:    "Promise",
				To:      prm.To,
				From:    prm.From,
				Promise: prm,
			}
			thisNetwork.SendMessage(prmMsg)
		case lrn := <-learnOut:
			fmt.Println("Sends learn out to network")
			lrnMsg := network.Message{
				Type:  "Learn",
				From:  lrn.From,
				Learn: lrn,
			}
			thisNetwork.SendMessageBroadcast(lrnMsg, []int{0, 1, 2})
		case response := <-decidedOut:
			resp := mp.Response{
				ClientID:  response.Value.ClientID,
				ClientSeq: response.Value.ClientSeq,
				Command:   response.Value.Command,
			}
			resMsg := network.Message{
				Type:     "Response",
				Response: resp,
			}
			fmt.Println("sending response to client", resp)
			thisNetwork.SendChannel <- resMsg
		case msg := <-thisNetwork.RecieveChannel:
			if msg.Type != "Heartbeat" {
				fmt.Println("I received msg", msg.Type)
			}
			switch {
			case msg.Type == "Heartbeat":
				hb := detector.Heartbeat{
					To:      msg.To,
					From:    msg.From,
					Request: msg.Request,
				}
				fmt.Println("Active fd deliver heartbeat")
				fd.DeliverHeartbeat(hb)
			case msg.Type == "Prepare":
				fmt.Println("Deliver prepare to acceptor")
				acceptor.DeliverPrepare(msg.Prepare)
			case msg.Type == "Promise":
				fmt.Println("Deliver promise to proposer")
				fmt.Println(msg.Promise)
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
				fmt.Println("Viiiiiiiiiiiiiiiiiiiiiiiiiiiiiiiiiiiiiii klllllllllllllllllllaaaaaaaaaaaaaaaaaaarteeeeeeeeeeeeeeeeeeeeee dettttttttttttttttttttttttttttttttttttttttt")
			}
		}

	}
}

// error check to save time
func check(err error) {
	if err != nil {
		log.Print(err)
	}
}
