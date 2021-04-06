package main

import (
	detector "dat520/lab3/detector"
	mp "dat520/lab5/multipaxos"
	"dat520/lab5/network"
	"encoding/json"
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"net"
	"os"

	//"strconv"
	"strings"
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
		42,
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

	defaultNrOfServers := 3
	//1.2 Read the network config file as byte array
	byteVal, _ := ioutil.ReadAll(netconfigureFile)

	//initialize a netconfig struct to save the netconfig
	var netconf network.Netconf
	err = json.Unmarshal(byteVal, &netconf)
	check(err)

	//1.3 Now netconf has the info from json file, step 1 complete
	//Nødvendig for å hjelpe docker å finne rett node ut fra IP den får
	fmt.Println(netconf)
	addr, _ := net.InterfaceAddrs()
	fmt.Println("I AM IRON", addr, addr[1])
	for _, conns := range netconf.Nodes {
		addrrr := addr[1].String()
		splitAddr := strings.Split(addrrr, "/")
		if conns.IP == splitAddr[0] {
			fmt.Println("testconn", conns.IP)
			fmt.Println("splitted", splitAddr[0])
			*id = conns.ID
		} else {
			fmt.Println("not it")
		}

	}
	// step 2: initialize a empty network with channels

	thisNetwork, currConf, err := network.InitializeNetwork(netconf.Nodes, *id, defaultNrOfServers)
	check(err)
	fmt.Println(thisNetwork.Myself.ID)
	// step 2.1: now we have a network with tcp endpoints, all nodes present

	// step 3: Initialize LD

	// step 3.1: in order to Init Leaderdetector we need a list of all nodes present
	nodeIDList := []int{*id}
	for _, node := range thisNetwork.Nodes {
		if len(nodeIDList) <= defaultNrOfServers-1 {
			nodeIDList = append(nodeIDList, node.ID)

		}
	}

	fmt.Println("These are the nodes you are looking for.. ", nodeIDList)
	/* 	//as we dont have myself in the nodelist we need to append thatone too
	   	nodeIDList = append(nodeIDList, thisNetwork.Myself.ID) */

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
	acceptorList := currConf.Acceptors
	acceptorList = append(acceptorList, *acceptor)

	//step 4.7 INIT learner
	decidedOut := make(chan mp.DecidedValue, 2000000) //send values that has been learned
	learner := mp.NewLearner(thisNetwork.Myself.ID, len(nodeIDList), decidedOut)
	learnerList := currConf.Learners
	learnerList = append(learnerList, *learner)

	fmt.Printf("The acceptors are %v, and the learners are %v", acceptorList, learnerList)

	// Append learner and acceptors to list?
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

	nrOfNodes := []int{0, 1, 2}

	ldchange := ld.Subscribe()
	fmt.Println("Leader is", ld.Leader())
	fd.Start()
	//reconfChann := make(chan network.Message, 2000000)
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
			thisNetwork.SendMessageBroadcast(prpMsg, nrOfNodes)
		case acc := <-acceptOut:
			fmt.Println("Sends accept out to network")
			accMsg := network.Message{
				Type:   "Accept",
				From:   acc.From,
				Accept: acc,
			}
			thisNetwork.SendMessageBroadcast(accMsg, nrOfNodes)
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
			thisNetwork.SendMessageBroadcast(lrnMsg, nrOfNodes)
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
			proposer.IncrementAllDecidedUpTo()
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
			case msg.Type == "reconf":
				fmt.Println("Reconf message recieved, reconfiguring with ", msg.Value.Command, " servers")
				//newNumber, _ := strconv.Atoi(msg.Value.Command)
				//proposer.DeliverConfig(msg)
				//Gjennomfør en Reconf(C) metode, hvor C er den nye configen.

				//fmt.Println("the old list of nodes", nodeIDList)

				/* nodeIDList = []int{*id}
				for _, node := range thisNetwork.Nodes {
					if (*id == node.ID){
						fmt.Println("are these the same?", *id, node.ID)
						continue
					} else if len(nodeIDList) <= newNumber-1 {
						fmt.Println(node.ID, newNumber)
						nodeIDList = append(nodeIDList, node.ID)
					}
				}
				fmt.Println("the new list of nodes", nodeIDList)

				nrOfNodes = []int{}
				for i := range nodeIDList{
					nrOfNodes = append(nrOfNodes, i)
				}
				fmt.Println("new nodelist", nrOfNodes)

				defaultNrOfServers = newNumber */

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
