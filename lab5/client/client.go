package main

import (
	"bufio"
	mp "dat520/lab5/multipaxos"
	network "dat520/lab5/network"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net"
	"os"
	"strconv"
)

func main() {

	// information about the network servers
	/* hardcoded := []string{
		"pitter14.ux.uis.no" + ":" + strconv.Itoa(20043),
		"pitter16.ux.uis.no" + ":" + strconv.Itoa(20043),
		"pitter3.ux.uis.no" + ":" + strconv.Itoa(20043),
	} */
	//nodeIDs := []int{0, 1, 2}

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

	nodes := []string{}
	for i, node := range netconfServers.Nodes {
		fmt.Println(i, node.Port)
		addr := node.IP + ":" + strconv.Itoa(node.Port)

		nodes = append(nodes, addr)

	}

	// Dial connection to the servers
	receiveChan := make(chan network.Message, 2000000)
	connections := Dial(nodes, receiveChan)

	seq := 0

	fmt.Println("Starting client on ip: " + ip.String())
	// Go function to receive input from terminal
	go func() {
		for {
			reader := bufio.NewReader(os.Stdin)

			fmt.Print("--> ")
			text, _ := reader.ReadString('\n')

			text = text[:len(text)-1]
			newVal := mp.Value{ClientID: ip.String(), ClientSeq: seq,
				Command: text}
			valMsg := network.Message{
				Type:  "Value",
				Value: newVal,
			}
			SendMessage(connections, valMsg)
		}
	}()
	for {
		msg := <-receiveChan
		if msg.Response.ClientSeq == seq {
			seq++
		}
		fmt.Println("Melding fra server. Client ID: " + msg.Response.ClientID + ", client seq: " + fmt.Sprint(msg.Response.ClientSeq) + ", client command: " + msg.Response.Command)
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
