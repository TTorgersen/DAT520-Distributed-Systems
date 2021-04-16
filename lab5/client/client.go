package main

import (
	"bufio"
	"dat520/lab5/bank"
	mp "dat520/lab5/multipaxos"
	network "dat520/lab5/network"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net"
	"os"
	"strconv"
	"strings"
)

var (
	myID string
	seq  = 0
	automsg = false
	//connections = make(map[string]*net.TCPConn)
	//delay       = 3 * time.Second
)

func main() {

	netconfigureFile, err := os.Open("../app/netconf.json")
	if err != nil {
		log.Print(err)
		return
	}
	defer netconfigureFile.Close()

	defaultNrOfServers := 3
	fmt.Println("NrofServers, ", defaultNrOfServers)

	//1.2 Read the network config file as byte array
	byteVal, _ := ioutil.ReadAll(netconfigureFile)

	//initialize a netconfig struct to save the netconfig
	var netconf network.Netconf
	err = json.Unmarshal(byteVal, &netconf)
	if err != nil {
		log.Print(err)
		return
	}

	// splitting netconf in servers and clients
	var netconfServers network.Netconf
	netconfServers.Nodes = netconf.Nodes[:6]
	netconfServers.Myself = netconf.Myself

	var netconfClients network.Netconf
	netconfClients.Nodes = netconf.Nodes[6:]
	netconfClients.Myself = netconf.Myself

	// Find the ip and port for this client
	ip, _ := GetOutboundIP()
	myID = ip.String()

	nodes := []string{}
	for i, node := range netconfServers.Nodes {
		fmt.Println(i, node.Port)
		addr := node.IP + ":" + strconv.Itoa(node.Port)

		nodes = append(nodes, addr)

	}

	// Dial connection to the servers
	receiveChan := make(chan network.Message, 2000000)
	connections := Dial(nodes, receiveChan)

	//seq := 0

	fmt.Println("Starting client on ip: " + ip.String())
	// Go function to receive input from terminal
	go func() {
		scanner := bufio.NewScanner(os.Stdin)
		for {

			//reader := bufio.NewReader(os.Stdin)
			fmt.Println("Enter command: Operation + Amount + Account")
			fmt.Println("Valid operation: balance, deposit, withdraw")
			fmt.Println("----------------------------")
			fmt.Println("Enter 'starttest' or 'stoptest' for testing")
			scanner.Scan()
			//fmt.Print("--> ")
			//text, _ := reader.ReadString('\n')
			input := scanner.Text()
			leaderQ := false // COMMAND REQUIRES leader with Space after leader
			if input == "leader " || input == "showhb " {
				leaderQ = true
			}
			if len(input) > 6 {

				

				if leaderQ || strings.Compare(input[0:7],"reconf ") == 0 {
					//input = input[:len(input)-1]
					confNr := input[7:]
					myVal := mp.Value{
						ClientID:   input,
						ClientSeq:  seq,
						AccountNum: 0,
						Command:    confNr}
					fmt.Println(myVal)
					valMsg := network.Message{
						Type:  "Value",
						Value: myVal,
					}
					fmt.Println("Value: ", myVal)
					fmt.Println("Valmsg", valMsg)
					fmt.Println("Sending reconf msg", valMsg.Value)
					SendMessage(connections, valMsg)
					fmt.Println("Sending reconf msg", valMsg.Value.ClientID)
				} else if strings.Compare("stoptest", input[0:8]) == 0{
					fmt.Println("Starting test")
					myVal3 := mp.Value{
						Command: "stoptest",
					}
					testMsg := network.Message{
						Type: "test", 
						Value: myVal3,
					}
					SendMessage(connections, testMsg)
				} else if strings.Compare("deposit ", input[0:8]) == 0 {
					fmt.Println("im in")
					val, err := inputToValue(input)

					fmt.Println("sending normal val", val)
					if err != nil {
						fmt.Println(err)
						continue
					}
					valMsg := network.Message{
						Type:  "Value",
						Value: val,
					}
					SendMessage(connections, valMsg)
					
				}else if strings.Compare("starttest", input[0:9]) == 0{
					fmt.Println("Starting test")
					myVal2 := mp.Value{
						Command: "starttest",
					}
					testMsg := network.Message{
						Type: "test", 
						Value: myVal2,
					}
					SendMessage(connections, testMsg)
				
				}else if strings.Compare("withdraw ", input[0:9]) == 0 {
						fmt.Println("im in")
						val, err := inputToValue(input)
	
						fmt.Println("sending normal val", val)
						if err != nil {
							fmt.Println(err)
							continue
						}
						valMsg := network.Message{
							Type:  "Value",
							Value: val,
						}
						SendMessage(connections, valMsg)
				} else {
					fmt.Println("Invalid input.")
					fmt.Println()
				}
			} else {
				fmt.Println("Invalid input.")
				fmt.Println() /*  else {
					val, err := inputToValue(input)

					fmt.Println("sending normal val", val)
					if err != nil {
						fmt.Println(err)
						continue
					}
					valMsg := network.Message{
						Type:  "Value",
						Value: val,
					}
					SendMessage(connections, valMsg)
				} */
			}
		}
		/*
			text = text[:len(text)-1]
			newVal := mp.Value{ClientID: ip.String(), ClientSeq: seq,
				Command: text}
			valMsg := network.Message{
				Type:  "Value",
				Value: newVal,
			}
			SendMessage(connections, valMsg)
		} */
	}()
	for {

		msg := <-receiveChan
		//fmt.Println(msg.Response)
		if msg.Response.ClientSeq == seq {
			seq++
			fmt.Println(msg.Response)
			if automsg{
				sendANewMsg(connections)
			}
			//fmt.Println("Skal egentlig printe dette også")
		}
		//fmt.Println("Melding fra server. Client ID: " + msg.Response.ClientID + ", client seq: " + fmt.Sprint(msg.Response.ClientSeq) + ", client command: " + msg.Response.Command)
	}
}

func GetOutboundIP() (net.IP, int) {
	conn, err := net.Dial("udp", "8.8.8.8:80")
	if err != nil {
		log.Fatal(err)
	}
	defer conn.Close()

	localAddr := conn.LocalAddr().(*net.UDPAddr)

	return localAddr.IP, localAddr.Port
}

func Dial(addresses []string, receive chan network.Message) map[int]*net.TCPConn {
	connection := make(map[int]*net.TCPConn)
	for i, addres := range addresses {
		a, _ := net.ResolveTCPAddr("tcp", addres)
		conn, err := net.DialTCP("tcp", nil, a)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Fatal error: %s\n", err)
			continue
		}
		connection[i] = conn
		go Listen(conn, receive)
	}
	return connection
}

func Listen(conn *net.TCPConn, receive chan network.Message) {
	buffer := make([]byte, 1024, 1024)
	for {
		len, err := conn.Read(buffer[0:])
		if err != nil {
			continue
		}

		message := network.Message{}
		err = json.Unmarshal(buffer[:len], &message)
		if err != nil {
			fmt.Println(err)
		}

		receive <- message
	}
}

func sendANewMsg(connections map[int]*net.TCPConn){
	input := "deposit 100 1"
	val, err := inputToValue(input)

				fmt.Println("sending normal val", val)
				if err != nil {
					fmt.Println(err)
				}
				valMsg := network.Message{
					Type:  "Value",
					Value: val,
				}
				SendMessage(connections, valMsg)

}


// Client send message to all network nodes
func SendMessage(connections map[int]*net.TCPConn, message network.Message) {
	messageByte, err := json.Marshal(message)
	for i, conn := range connections {
		_, err = conn.Write(messageByte)
		if err != nil {
			conn.Close()
			delete(connections, i)
		}
	}
}
func inputToValue(input string) (val mp.Value, err error) {
	commands := strings.Fields(input)
	if len(commands) == 2 {
		accountNum, _ := strconv.Atoi(commands[1])
		if commands[0] == "balance" {
			return mp.Value{
				ClientID:   myID,
				ClientSeq:  seq,
				Noop:       false,
				AccountNum: accountNum,
				Txn: bank.Transaction{
					Op:     bank.Operation(0),
					Amount: 0,
				},
			}, nil
		}
	}
	if len(commands) == 3 {
		amount, _ := strconv.Atoi(commands[1])
		accountNum, _ := strconv.Atoi(commands[2])
		if commands[0] == "deposit" {
			return mp.Value{
				ClientID:   myID,
				ClientSeq:  seq,
				Noop:       false,
				AccountNum: accountNum,
				Txn: bank.Transaction{
					Op:     bank.Operation(1),
					Amount: amount,
				},
			}, nil
		}
		if commands[0] == "withdraw" {
			return mp.Value{
				ClientID:   myID,
				ClientSeq:  seq,
				Noop:       false,
				AccountNum: accountNum,
				Txn: bank.Transaction{
					Op:     bank.Operation(2),
					Amount: amount,
				},
			}, nil
		}
	}
	return val, nil
}
