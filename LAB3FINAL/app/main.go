package main

import (
	detector "dat520/LAB3FINAL/detectors"
	"dat520/LAB3FINAL/network"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"time"
	mp "dat520/lab4/multipaxos"
)

func main() {
	fmt.Println("Main application")

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
	thisNetwork, err := network.InitializeNetwork(netconf.Nodes, netconf.Myself)
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
	hbSend := make(chan detector.Heartbeat, 100)
	fd := detector.NewEvtFailureDetector(thisNetwork.Myself.ID, nodeIDList, ld, 10*time.Second, hbSend)

	// step 3 and 4 complete, now we have a failiure detector
	fmt.Println(fd)
	// step 5: Initialize connections
	// step 6: start server
	fmt.Println("Trying to init connections")
	thisNetwork.InitializeConnections()
	thisNetwork.StartServer()
	fmt.Println("Started server, we good")

	ldchange := ld.Subscribe()
	fmt.Println("Leader is", ld.Leader())
	fd.Start()


	rcvClient := make(chan mp.Value, 100)
	

	for {

		select {
		case newLeader := <-ldchange:
			fmt.Println("A NEW LEADER HAS BEEN CHOSEN, ALL HAIL LEADER ", newLeader)
			fmt.Printf("\nSuspected nodes at ld: %v\n", ld.Suspected)
		case hb := <-hbSend:
			sendHBMessage := network.Message{
				To:      hb.To,
				From:    hb.From,
				Request: hb.Request,
			}
			thisNetwork.SendChannel <- sendHBMessage

		case recieveHB := <-thisNetwork.RecieveChannel:
			hb := detector.Heartbeat{
				To:      recieveHB.To,
				From:    recieveHB.From,
				Request: recieveHB.Request,
			}
			fd.DeliverHeartbeat(hb)
		}

	}
}

// error check to save time
func check(err error) {
	if err != nil {
		log.Print(err)
	}
}
