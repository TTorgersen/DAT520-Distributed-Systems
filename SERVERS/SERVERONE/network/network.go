package network

import (
	"encoding/json"
	"fmt"
	"log"
	"net"
	"strconv"
	"strings"
	"sync"
)

//Netconf Network config struct
type Netconf struct {
	Myself int
	Nodes  []Node
}

//Node struct
type Node struct {
	ID        int
	IP        string
	Port      int
	TCPaddr   *net.TCPAddr
	TCPListen *net.TCPListener //what thisNode uses

}

//Network struct is the basis of our network
type Network struct {
	Myself         Node                 // my node
	Nodes          []Node               // all nodes in network
	Connections    map[int]*net.TCPConn // all connections to all nodes in network
	RecieveChannel chan Message
	SendChannel    chan Message
}

//Message Struct for sending and recieving across network
type Message struct {
	Type    string // heartbeat
	To      int    //nodeid
	From    int    //nodeid
	Request bool   // heartbeat true/false

}

//InitializeNetwork creates a empty network with channels ready
func InitializeNetwork(nodes []Node, Myself int) (network Network, err error) {

	// creates a recieving and send channel
	reciveChann := make(chan Message, 65000)
	sendChann := make(chan Message, 65000)

	// create a network with empty nodes and channels
	network = Network{
		Nodes:          []Node{},
		Connections:    map[int]*net.TCPConn{},
		RecieveChannel: reciveChann,
		SendChannel:    sendChann,
	}

	// for each node, add tcpNetwork

	for _, node := range nodes {
		if node.ID == Myself {
			fmt.Printf("YOU are node number %v\n", node.ID)
			network.Myself = node
			//address of myself
			address := network.Myself.IP + ":" + strconv.Itoa(network.Myself.Port)
			// adding the  tcp endpoint with resolveTCPaddr
			network.Myself.TCPaddr, err = net.ResolveTCPAddr("tcp", address)
		} else {
			// node is not myself, address of node
			address := node.IP + ":" + strconv.Itoa(node.Port)
			tcpEndpoint, _ := net.ResolveTCPAddr("tcp", address)
			network.Nodes = append(network.Nodes, Node{
				ID:      node.ID,
				IP:      node.IP,
				Port:    node.Port,
				TCPaddr: tcpEndpoint,
			})
		}

	}
	return network, err

}

//InitializeConnections will try to initiate connections with the servers
//starting tcp server and initiate contact
func (n *Network) InitializeConnections() (err error) {
	// we loop all nodes and try to dial them with dialTCP
	for _, node := range n.Nodes {
		TCPDial, err := net.DialTCP("tcp", nil, node.TCPaddr)
		if check(err) {
			continue
		} else {
			n.Connections[node.ID] = TCPDial
			fmt.Printf("Dial via tcp to node %v success\n", node.TCPaddr)
		}

		// create a separate go routine in order to handle dial connections
		go n.ListenForConnection(TCPDial)
	}
	return err
}

// error check to save time
func check(err error) (errors bool) {
	if err != nil {
		log.Print(err)
		return true
	}
	return false

}

//ListenForConnection shall be initialized in a separate go routine
// in order
func (n *Network) ListenForConnection(TCPConnection *net.TCPConn) (err error) {

	// defers closing connection until the end
	defer n.CloseConn(TCPConnection)

	buffer := make([]byte, 1024, 1024)

	//etarnal for loop to handle listening to connections
	for {

		len, _ := TCPConnection.Read(buffer[0:])
		message := new(Message)
		fmt.Println("MELDING", message)

		err = json.Unmarshal(buffer[0:len], &message)
		if message.Type != "Heartbeat" {
			fmt.Print("ERROR")
		}
		n.RecieveChannel <- *message

	}
	return err
}

//Mutex to lock and unlock go routine
var Mutex = &sync.Mutex{}

//CloseConn tries to close the connection
func (n *Network) CloseConn(TCPConnection *net.TCPConn) {
	TCPConnection.Close()
	fmt.Println("Network is closing the connection from", TCPConnection.RemoteAddr())

	NodeID := n.findRemoteAdrress(TCPConnection)

	// locks go routine to prevent errors
	Mutex.Lock()
	delete(n.Connections, NodeID)
	Mutex.Unlock()
}

//finds the node id based on the remote address it got in
func (n *Network) findRemoteAdrress(TCPConnection *net.TCPConn) (NodeID int) {

	RemoteSocket := TCPConnection.RemoteAddr()
	RemoteIPPort := strings.Split(RemoteSocket.String(), ":")
	RemotePort := RemoteIPPort[0]
	//portInt, _ := strconv.Atoi(RemotePort)
	for _, node := range n.Nodes {
		print(node.IP, RemotePort)
		if node.IP == RemotePort {
			fmt.Println("found node from ip", node.ID, RemotePort)
			return node.ID

		}
	}
	fmt.Println("CANT FIND NODE ID", RemoteSocket)
	return -1
}

//StartServer starts a tcp listener on application host
func (n *Network) StartServer() (err error) {
	TCPListn, err := net.ListenTCP("tcp", n.Myself.TCPaddr)
	fmt.Println("starting TCP server on node ", n.Myself.ID, n.Myself.TCPaddr)
	check(err)
	// sets this applications listening post
	n.Myself.TCPListen = TCPListn

	go func() { // listening for TCP connections
		defer TCPListn.Close()
		for {
			//accepting a tcp call and returning a new connection
			TCPaccept, err := TCPListn.AcceptTCP()
			check(err)

			// find out which node is sending it
			NodeID := n.findRemoteAdrress(TCPaccept)

			Mutex.Lock()
			n.Connections[NodeID] = TCPaccept
			Mutex.Unlock()
			fmt.Println("Accepted TCP from node ", NodeID)

			go n.ListenForConnection(TCPaccept)
		}

	}()
	go func() { // Listens on sendCHannel for messages
		for {
			message := <-n.SendChannel
			err := n.SendMessage(message)
			check(err)
		}

	}()
	return err
}

//SendMessage sends a message to the desired recipient
func (n *Network) SendMessage(message Message) (err error) {
	if message.To == n.Myself.ID {
		n.RecieveChannel <- message
		return
	}

	messageByte, err := json.Marshal(message)
	check(err)
	remoteConn := n.Connections[message.To]
	if remoteConn == nil {
		return fmt.Errorf("No connection to ", message.To)
	}
	_, err = n.Connections[message.To].Write(messageByte)
	if check(err) {
		n.CloseConn(n.Connections[message.To])
		return err
	}
	return err

}
