package main

import (
	//detector "dat520/lab3/detector"
	fd "dat520/lab3/failuredetector"
	ld "dat520/lab3/leaderdetector"
	bh "dat520/lab5/bankhandler"
	mp "dat520/lab5/multipaxos"
	"dat520/lab5/network"
	"encoding/json"
	"flag"
	"fmt"
	"io/ioutil"
	

	//"log"
	"net"
	"os"
	"sort"

	"strconv"
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
		5000,
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

	alive := true
	showHB := false
	ValuesPrePaxos:= []mp.Value{}
	batchingDelay := 500
	ValuesPrePaxos = nil
	completedPaxos := 0.0
	var start time.Time
	 
	
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

	// splitting netconf in servers and clients
	var netconfServers network.Netconf
	netconfServers.Nodes = netconf.Nodes[:6]
	netconfServers.Myself = netconf.Myself

	var netconfClients network.Netconf
	netconfClients.Nodes = netconf.Nodes[6:]
	netconfClients.Myself = netconf.Myself

	var myaddress network.Node
	//1.3 Now netconf has the info from json file, step 1 complete
	//Nødvendig for å hjelpe docker å finne rett node ut fra IP den får
	fmt.Println(netconfServers)
	addr, _ := net.InterfaceAddrs()
	for _, conns := range netconfServers.Nodes {
		addrrr := addr[1].String()
		splitAddr := strings.Split(addrrr, "/")
		if conns.IP == splitAddr[0] {
			*id = conns.ID
			myaddress = netconfServers.Nodes[conns.ID]

		} else {
			fmt.Println("not it")
		}

	}

	selfAddress, err := net.ResolveTCPAddr("tcp", myaddress.IP+":"+strconv.Itoa(myaddress.Port))
	check(err)
	fmt.Println("selfadress: ", selfAddress)
	selfNode := network.Node{
		ID:      *id,
		IP:      selfAddress.IP.String(),
		Port:    selfAddress.Port,
		TCPaddr: selfAddress,
	}

	nodes := []network.Node{}
	for i, node := range netconfServers.Nodes {
		fmt.Println(i, node.Port)
		addr := node.IP + ":" + strconv.Itoa(node.Port)
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


	nodeIDList := []int{*id}
	for _, node := range thisNetwork.Nodes {
		nodeIDList = append(nodeIDList, node.ID)
	}

	// ______________________________ linje 103 oystein
	sort.Ints(nodeIDList)
	fmt.Println("These are the nodes you are looking for.. ", nodeIDList[:defaultNrOfServers])

	// Set up leader and failuredetector
	hbSend := make(chan fd.Heartbeat, 16)

	timerChan := make(chan struct{})

	leaderdetector := ld.NewMonLeaderDetector(nodeIDList[:defaultNrOfServers])
	failuredetector := fd.NewEvtFailureDetector(*id, nodeIDList[:defaultNrOfServers], leaderdetector, time.Duration(*delay)*time.Millisecond, hbSend)

	// Set up multipaxos
	//INIT MULTIPAXOS proposer
	prepareOut := make(chan mp.Prepare, 2000000) //sendonly channel send prepare
	acceptOut := make(chan mp.Accept, 2000000)   //Incoming accept
	//adu -1 is initially
	proposer := mp.NewProposer(thisNetwork.Myself.ID, len(nodeIDList[:defaultNrOfServers]), -1, leaderdetector, prepareOut, acceptOut)

	//Step 4.6 INIT acceptor
	promiseOut := make(chan mp.Promise, 2000000) //send promises to other nodes
	learnOut := make(chan mp.Learn, 2000000)     //send learn to other nodes
	acceptor := mp.NewAcceptor(thisNetwork.Myself.ID, promiseOut, learnOut)
	//step 4.7 INIT learner
	decidedOut := make(chan mp.DecidedValue, 2000000) //send values that has been learned
	learner := mp.NewLearner(thisNetwork.Myself.ID, len(nodeIDList[:defaultNrOfServers]), decidedOut)

	responseOut := make(chan mp.Response, 2000000)
	bankhandler := bh.NewBankHandler(responseOut, proposer)
	// start up the network
	thisNetwork.Dial()
	thisNetwork.StartServer()

	fmt.Println("Asking for status update")
	statusRecieved := false
	if len(thisNetwork.Connections) > 0 {
		updateMsg := network.Message{
			Type: "statusUpdate",
			From: *id,
		}
		thisNetwork.SendMessageBroadcast(updateMsg, []int{0, 1, 2, 3, 4, 5, 6})
	} else {
		fmt.Println("No servers in reach, continuing")
		statusRecieved = true
	}
	for {
		if statusRecieved {
			break
		}
		select {
		case msg := <-thisNetwork.RecieveChannel:
		//	fmt.Println("message recieved", msg.Type)

			if msg.Type == "statusResponse" {
				fmt.Println("status response recieved, defaultNrOfServers: ", msg.From)
				defaultNrOfServers = msg.From
				fmt.Println("Setting crnd ", msg.Rnd)
				proposer.SetCrnd(msg.Rnd)
				fmt.Println("Setting Adu proposer ", msg.Adu)
				proposer.SetSlot(msg.Adu)
				
				fmt.Println("Setting Adu BH ", msg.Adu)
				bankhandler.SetSlot(msg.Adu)
				
				fmt.Println("Setting Bank accounts ", msg.BankAccounts)
				bankhandler.SetBankAccounts(msg.BankAccounts)
				
				fmt.Println("Setting Lrn slot ", msg.LrnSlots)
				learner.SetLearnSlot(msg.LrnSlots)
				
				fmt.Println("Setting lrn sent ", msg.LrnSent)
				learner.SetLearnSent(msg.LrnSent)
				
				fmt.Println("Setting acceptor slots")
				fmt.Println("the slots", msg.AccSlots)
				acceptor.SetSlot(msg.AccSlots)
				
				fmt.Println("Crnd Recieved: ", proposer.Crnd())
				statusRecieved = true
			}
		}

	}
	//for conn
	//write statusupdate

	//defaultNrOfServers = answer
	go startTimer(batchingDelay, timerChan)

	if *id >= defaultNrOfServers {
		alive = false
		fmt.Println(alive)
		fmt.Println(*id, " not connected waiting for signal")

	}
	ldchange := leaderdetector.Subscribe()

	if alive {
		failuredetector.Start()
		// Start multipaxos
		proposer.Start()
		acceptor.Start()
		learner.Start()

		fmt.Println("Starting server: ", selfAddress, " With id: ", *id)

		//go subscribePrinter(leaderdetector.Subscribe())
	}
	for {
		
		time.Sleep(100 * time.Millisecond)
		
		select {
		case newL := <-ldchange:
			if alive {
				fmt.Println("new leader is ", newL)
			}
		// make heartbeat message and send to receiver
	case  <- timerChan:
		
		if ValuesPrePaxos != nil && leaderdetector.CurrentLeader == thisNetwork.Myself.ID{
		BatchList := mp.ValueList{
			Noop: false,
			ListOfRequests: ValuesPrePaxos,
		}	
		fmt.Println("", batchingDelay, " seconds has passed")
		fmt.Println("Number of requests in queue: ", len(ValuesPrePaxos))

		proposer.DeliverClientValue(BatchList)
		ValuesPrePaxos = nil
		}
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
			//fmt.Println("Sends prepare out to network: ", prp.String())
			prpMsg := network.Message{
				Type:    "Prepare",
				From:    prp.From,
				Prepare: prp,
			}
			//fmt.Println("Trying to send message to ", nodeIDList[:defaultNrOfServers])
			thisNetwork.SendMessageBroadcast(prpMsg, nodeIDList[:defaultNrOfServers])
		// make accept message and send to network
		case acc := <-acceptOut:
			//fmt.Println("Sends accept out to network: ", acc.String())
			accMsg := network.Message{
				Type:   "Accept",
				From:   acc.From,
				Accept: acc,
			}
			thisNetwork.SendMessageBroadcast(accMsg, nodeIDList[:defaultNrOfServers])
		// make promise message and send to receiver
		case prm := <-promiseOut:
			//fmt.Println("Sends promise out to network: ", prm.String())
			prmMsg := network.Message{
				Type:    "Promise",
				To:      prm.To,
				From:    prm.From,
				Promise: prm,
			}
			thisNetwork.SendMessage(prmMsg)
		// make learn message and send to network
		case lrn := <-learnOut:
			//fmt.Println("Sends learn out to network: ", lrn.String())
			lrnMsg := network.Message{
				Type:  "Learn",
				From:  lrn.From,
				Learn: lrn,
			}
			thisNetwork.SendMessageBroadcast(lrnMsg, nodeIDList[:defaultNrOfServers])
		// make response message and send to client
		case decidedList := <-decidedOut:
			//fmt.Println("Paxos decided",decidedList)
			for _, decided := range(decidedList.Value.ListOfRequests){
			//		fmt.Println("This has been decided ", decided.Value)
			if len(decided.ClientID) >= 6 {

				if decided.ClientID[0:7] == "reconf " {

					defaultNrOfServers, _ = strconv.Atoi(decided.ClientID[7:])
					fmt.Println("Reconfigure request decided, Stopping servers, new number: ", defaultNrOfServers)
					// WE MUST INCREMENT SLOT ANYWAYS
					proposer.IncrementAllDecidedUpTo()
					// THIS IS NEW
					
					learner.Stop()
					proposer.Stop()
					acceptor.Stop()
					currL := leaderdetector.CurrentLeader
					failuredetector.Stop()
					alive = false // just testing for fun
					rMsg := network.Message{
						Type: "reconf",
						From: defaultNrOfServers, 
						Rnd:          proposer.Crnd(),
						Adu:          proposer.GetSlot(),
						BankAccounts: bankhandler.GetBankAccounts(),
						LrnSlots:     learner.GetLearnSlot(),
						LrnSent:      learner.GetLearnSent(),
						AccSlots: acceptor.GetSlot(),
						
						//we just use "from " because it is int
						// HER KAN VI OGSÅ SENDE CURRENT DATA
					}
					//fmt.Println("Sending a broadcast message to ", nodeIDList[:defaultNrOfServers])
					if *id == currL {
						thisNetwork.SendMessageBroadcast(rMsg, nodeIDList[:defaultNrOfServers])
						fmt.Println("I am leader, I send reconf msg to servers")

					}

					continue
				} else {
					d := mp.OneDecidedValue{
						SlotID: decidedList.SlotID,
						Value: decided,
					}
					//fmt.Println("Sends decided value to bankhandler with slotID "+fmt.Sprint(d.SlotID)+", and value: ", d.Value.String())
					completedPaxos += 1.0
					bankhandler.HandleDecidedValue(d)
				}
			} else {
				d := mp.OneDecidedValue{
					SlotID: decidedList.SlotID,
					Value: decided,
				}
				//fmt.Println("Sends decided value to bankhandler with slotID "+fmt.Sprint(decided.SlotID)+", and value: ", decided.Value.String())
				//fmt.Println("Sends decided value to bankhandler with slotID "+fmt.Sprint(d.SlotID)+", and value: ", d.Value.String())
				completedPaxos += 1.0
				bankhandler.HandleDecidedValue(d)
			}}
		case response := <-responseOut:
			//fmt.Println("A VALUE HAS BEEN DECIDED ",response)
			resp := mp.Response{
				ClientID:  response.ClientID,
				ClientSeq: response.ClientSeq,
				TxnRes:    response.TxnRes,
			}
			resMsg := network.Message{
				Type:     "Response",
				Response: resp,
				ClientIP: response.ClientID,
			}
			// Y DO
			//proposer.IncrementAllDecidedUpTo()

			fmt.Println("Sends response to client", resp.TxnRes.Balance)
			thisNetwork.SendMessage(resMsg)
		/* 	if len(response.Value.Command) > 6 {
			if response.Value.Command[0:7] == "reconf " {
				defaultNrOfServers, _ = strconv.Atoi(response.Value.Command[7:])
				fmt.Println("Reconfigure request decided, Stopping servers, new number: ", defaultNrOfServers)
				learner.Stop()
				proposer.Stop()
				acceptor.Stop()
				failuredetector.Stop()
				alive = false // just testing for fun
				rMsg := network.Message{
					Type: "reconf",
					From: defaultNrOfServers, //we just use "from " because it is int
					// HER KAN VI OGSÅ SENDE CURRENT DATA
				}
				fmt.Println("Sending a broadcast message to ", nodeIDList)
				thisNetwork.SendMessageBroadcast(rMsg, nodeIDList)

				//continue
			}
		}
		*/

		// message on network to on self
		case msg := <-thisNetwork.RecieveChannel:
			
			if msg.Type == "test" {
				fmt.Println("commando", msg.Value.Command)
				if (msg.Value.Command == "starttest") {
					fmt.Println("Starting timer ...")
					start = time.Now()
					completedPaxos = 0
				} 
				if(msg.Value.Command == "stoptest"){
					elapsed := float64(time.Since(start)/1000000000)
					fmt.Printf("Stopped test with %v seconds elapsed and %v tests executed.", elapsed, completedPaxos)
					avgReq :=  fmt.Sprintf("the num %.2f", completedPaxos/elapsed)
					fmt.Println("Resulting in ", avgReq, " per second.")
				}	
				
				
			}
			if msg.Type == "statusUpdate" {
				if *id == leaderdetector.CurrentLeader {
					fmt.Println("Responding to statusUpdate")
					responseMsg := network.Message{
						Type:         "statusResponse",
						From:         defaultNrOfServers,
						To:           msg.From,
						Rnd:          proposer.Crnd(),
						Adu:          proposer.GetSlot(),
						BankAccounts: bankhandler.GetBankAccounts(),
						LrnSlots:     learner.GetLearnSlot(),
						LrnSent:      learner.GetLearnSent(),
						AccSlots: 	  acceptor.GetSlot()	,	
					}
					fmt.Println()
					fmt.Println("Response message sent: ", responseMsg.Type, responseMsg.From, responseMsg.To, responseMsg.Rnd, responseMsg.BankAccounts, responseMsg.LrnSlots, responseMsg.LrnSent)
					thisNetwork.SendMessage(responseMsg)
				}
			}
			if msg.Type == "reconf" {
				fmt.Println("Reconfigure message recieved over network")
				defaultNrOfServers = msg.From
				if *id < defaultNrOfServers {
					//sleeping addon
					time.Sleep(time.Duration(thisNetwork.Myself.ID) * time.Millisecond)

					leaderdetector = ld.NewMonLeaderDetector(nodeIDList[:defaultNrOfServers])
					failuredetector = fd.NewEvtFailureDetector(*id, nodeIDList[:defaultNrOfServers], leaderdetector, time.Duration(*delay)*time.Millisecond, hbSend)

					alive = true
					fmt.Println("Reconfigure request recieved, starting up server ID: ", *id)

					learner = mp.NewLearner(thisNetwork.Myself.ID, defaultNrOfServers, decidedOut)
					proposer = mp.NewProposer(thisNetwork.Myself.ID, defaultNrOfServers, -1, leaderdetector, prepareOut, acceptOut)
					acceptor = mp.NewAcceptor(thisNetwork.Myself.ID, promiseOut, learnOut)
					bankhandler = bh.NewBankHandler(responseOut, proposer)
					
					
					fmt.Println("Setting crnd ", msg.Rnd)
					proposer.SetCrnd(msg.Rnd)
					fmt.Println("Setting Adu proposer ", msg.Adu)
					proposer.SetSlot(msg.Adu)
					
					fmt.Println("Setting Adu BH ", msg.Adu)
					bankhandler.SetSlot(msg.Adu)
					
					fmt.Println("Setting Bank accounts ", msg.BankAccounts)
					bankhandler.SetBankAccounts(msg.BankAccounts)
					
					fmt.Println("Setting Lrn slot ", msg.LrnSlots)
					learner.SetLearnSlot(msg.LrnSlots)
					
					fmt.Println("Setting lrn sent ", msg.LrnSent)
					learner.SetLearnSent(msg.LrnSent)
					
					fmt.Println("Setting acceptor slots")
					acceptor.SetSlot(msg.AccSlots)
					
					
					
					proposer.Start()
					acceptor.Start()
					learner.Start()

					fmt.Println("Starting server: ", selfAddress, " With id: ", *id)

					ldchange = leaderdetector.Subscribe()
					leaderdetector.ChangeLeader()
					failuredetector.Start()

					//thisNetwork.Dial()
					//showHB = true

					//go subscribePrinter(leaderdetector.Subscribe())
				} else {
					alive = false
				}
			}
			if alive {

				switch {
				case msg.Type == "Heartbeat":
					if showHB == true {
						fmt.Println(msg.Heartbeat)
					}
					//fmt.Println(msg.Heartbeat)
					failuredetector.DeliverHeartbeat(msg.Heartbeat)
				case msg.Type == "Prepare":
					acceptor.DeliverPrepare(msg.Prepare)
				case msg.Type == "Promise":
				//	fmt.Println("Deliver promise to proposer")
					proposer.DeliverPromise(msg.Promise)
				case msg.Type == "Accept":
				//	fmt.Println("Deliver accept to acceptor")
					acceptor.DeliverAccept(msg.Accept)
				case msg.Type == "Learn":
				//	fmt.Println("Deliver learn to learner")
					learner.DeliverLearn(msg.Learn)
				case msg.Type == "Value":

					if msg.Value.ClientID == "leader " {

						fmt.Println("Current leader leaderdetector: ", leaderdetector.CurrentLeader)
						fmt.Println("Current leader proposer: ", proposer.Leader())

					} else if msg.Value.ClientID == "showhb " {
						showHB = !showHB

					} else if msg.Value.ClientID == "status" {
						fmt.Println("Phase1 done?: ", proposer.Phase1())
						fmt.Println("Proposer slot ",proposer.GetSlot(), " Proposer round: ", proposer.Crnd())
						fmt.Println("connections: ", len(thisNetwork.Connections))
					} else {
					//	fmt.Println("Deliver value from client to proposer", msg.Value)
					// her samle requests

						ValuesPrePaxos = append(ValuesPrePaxos, msg.Value)
						//proposer.DeliverClientValue(msg.Value)
					}
				case msg.Type == "Responce":
					fmt.Println(msg.Response)
				}
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

func startTimer(t int, timerChan chan struct{}){
	timer := time.NewTimer(time.Duration(t) * time.Millisecond)
	<- timer.C
	timerChan <- struct{}{}
	startTimer(t, timerChan)
	
}
/*
	if alive {
		//start multiPax  // CHANGE JEG GJØR EN ENDRING commenter out dette fjerner multipaxos hypotese
		//proposer.Start()
		//acceptor.Start()
		//learner.Start()

		//fmt.Println("Leader is", ld.Leader())
		//fd.Start()
	}
	/* //reconfChann := make(chan network.Message, 2000000)
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
		case prp := <-prepareOut: // CHANGE
			fmt.Println("Sends prepare out to network", prp)
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
			fmt.Println("A VALUE HAS BEEN DECIDED ",response.Value)
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
			//update network config alive node

			if len(response.Value.Command) > 6 {
				if response.Value.Command[0:7] == "reconf " {
					defaultNrOfServers, _ = strconv.Atoi(response.Value.Command[7:])
					fmt.Println("Reconfigure request decided, Stopping servers, new number: ", defaultNrOfServers)
					learner.Stop()
					proposer.Stop()
					acceptor.Stop()
					fd.Stop()
					nrOfNodes = []int{}

					for i := 0; i < defaultNrOfServers; i++ {
						nrOfNodes = append(nrOfNodes, i)
					}

					rMsg := network.Message{
						Type: "reconf",
						From: defaultNrOfServers, //we just use "from " because it is int
					}
					fmt.Println("Sending a broadcast message to ", nrOfNodes)
					thisNetwork.SendMessageBroadcast(rMsg, nrOfNodes)

					continue
				}
			}
			thisNetwork.SendChannel <- resMsg
		case msg := <-thisNetwork.RecieveChannel:
			if msg.Type == "reconf" {
				fmt.Println("Reconfigure message recieved over network")
				defaultNrOfServers = msg.From
				if *id < defaultNrOfServers {

					alive = true
					fmt.Println("Reconfigure request recieved, starting up server ID: ", *id)
					ld = detector.NewMonLeaderDetector(nodeIDList[:defaultNrOfServers])
					fd = detector.NewEvtFailureDetector(thisNetwork.Myself.ID, nodeIDList[:defaultNrOfServers], ld, 10*time.Second, hbSend)

					learner = mp.NewLearner(thisNetwork.Myself.ID, defaultNrOfServers, decidedOut)
					proposer = mp.NewProposer(thisNetwork.Myself.ID, defaultNrOfServers, -1, ld, prepareOut, acceptOut)
					acceptor = mp.NewAcceptor(thisNetwork.Myself.ID, promiseOut, learnOut)

					proposer.Start()
					acceptor.Start()
					learner.Start()

					ldchange = ld.Subscribe()
					fmt.Println("Leader is", ld.Leader())
					fd.Start()
				} else {
					alive = false
				}
			}
			if alive {
				switch {
				case msg.Type == "Heartbeat":
					hb := detector.Heartbeat{
						To:      msg.To,
						From:    msg.From,
						Request: msg.Request,
					}
					//	fmt.Println("Active fd deliver heartbeat")
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
				}

				/* case msg.Type == "reconf":
				fmt.Println("Reconf message recieved, reconfiguring with ", msg.Value.Command, " servers") */
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

				defaultNrOfServers = newNumber

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
*/
