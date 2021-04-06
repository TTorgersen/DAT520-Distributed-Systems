package network

import (
	mp "dat520/lab5/multipaxos"
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
	Myself            Node                 // my node
	Nodes             []Node               // all nodes in network
	Connections       map[int]*net.TCPConn // all connections to all nodes in network
	ClientConnections []*net.TCPConn
	RecieveChannel    chan Message
	SendChannel       chan Message
}


//Will implement a reconfig method which determines how Config will update
type Config struct { //Will determine how the network works. How many servers etc..
	cfg int //which config is used, start with 1
	//MyConfig MyC //my own info
	Nodes             []Node
	Connections       map[int]*net.TCPConn //all server connections
	ClientConnections []*net.TCPConn       //the clients
	Acceptors         []mp.Acceptor        //How many acceeptors
	Learners          []mp.Learner         //the learners
}

//Message Struct for sending and recieving across network
type Message struct {
	Type         string          // heartbeat, accept, promise, prepare, learn, value, response
	To           int             //nodeid
	From         int             //nodeid
	Request      bool            // true = request, false = reply
	Accept       mp.Accept       //accept msg
	Promise      mp.Promise      //promise msg
	Prepare      mp.Prepare      //prepare msg
	Learn        mp.Learn        //learn msg
	Value        mp.Value        //value msg
	Response     mp.Response     //response msg
	Decidedvalue mp.DecidedValue //decidedvalue
	Reconf       mp.Reconf
	Alive		bool  //should node be alive or not
}

//InitializeNetwork creates a empty network with channels ready
func InitializeNetwork(nodes []Node, Myself int) (network Network, currConf Config, err error) {

	// creates a recieving and send channel
	reciveChann := make(chan Message, 2000000)
	sendChann := make(chan Message, 2000000)

	// create a network with empty nodes and channels
	network = Network{
		Nodes:             []Node{},
		Connections:       map[int]*net.TCPConn{},
		ClientConnections: []*net.TCPConn{}, //added this to check if it helps the Clientconnections
		RecieveChannel:    reciveChann,
		SendChannel:       sendChann,
	}


	// for each node, add tcpNetwork
	for _, node := range nodes {
			if node.ID == Myself {
				//fmt.Printf("YOU are node number %v\n", node.ID)
				network.Myself = node
				//address of myself
				address := network.Myself.IP + ":" + strconv.Itoa(network.Myself.Port)
				// adding the  tcp endpoint with resolveTCPaddr
				network.Myself.TCPaddr, err = net.ResolveTCPAddr("tcp", address)
			} else {
				// node is not myself, address of node
				fmt.Println("added foreign address")
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
	return network, currConf, err

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
	
				//fmt.Printf("Dial via tcp to node %v success\n", node.TCPaddr)
			}
			go n.ListenForConnection(TCPDial)

		// create a separate go routine in order to handle dial connections
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
		err = json.Unmarshal(buffer[0:len], &message)
		if check(err) {
			return err
		}
		n.RecieveChannel <- *message

	}
}

//Mutex to lock and unlock go routine
var Mutex = &sync.Mutex{}

//CloseConn tries to close the connection
func (n *Network) CloseConn(TCPConnection *net.TCPConn) {
	TCPConnection.Close()
	//fmt.Println("Network is closing the connection from", TCPConnection.RemoteAddr())

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
	fmt.Println("the nodes in remote", n.Nodes)
	for _, node := range n.Nodes {
		print(node.IP, RemotePort)
		if node.IP == RemotePort {
			fmt.Println("found node from ip", node.ID, RemotePort)
			return node.ID

		}
	}
	//fmt.Println("CANT FIND NODE ID", RemoteSocket)
	return -1
}

//StartServer starts a tcp listener on application host
func (n *Network) StartServer() (err error) {
	TCPListn, err := net.ListenTCP("tcp", n.Myself.TCPaddr)
	//fmt.Println("starting TCP server on node ", n.Myself.ID, n.Myself.TCPaddr)
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
			if NodeID == -1 {
				fmt.Println("A new client has connected")
				n.ClientConnections = append(n.ClientConnections, TCPaccept)

				fmt.Println("Client connections", n.ClientConnections)
				fmt.Println("Servers connections", n.Connections)
			} else {
				Mutex.Lock()
				n.Connections[NodeID] = TCPaccept
				Mutex.Unlock()
				//fmt.Println("Accepted TCP from node ", NodeID)
			}
			go n.ListenForConnection(TCPaccept)
		}

	}()
	go func() { // Listens on sendCHannel for messages
		for {
			//message := <-n.SendChannel
			select {
			case message := <-n.SendChannel:
				switch {
				case message.Type == "Response":
					for _, conns := range n.ClientConnections {
						messageByte, err := json.Marshal(message)
						if err != nil {
							log.Print(err)
							continue
						}
						_, err = conns.Write(messageByte)
						if err != nil {
							log.Print(err)
						}
					}
				case message.Type != "Response":
					err := n.SendMessage(message)
					if err != nil {
						log.Print(err)
					}
				}
			}
		}

	}()
	return err
}

//SendCommand to other modules
func (n *Network) SendMessageBroadcast(message Message, destination []int) {
	for _, destID := range destination {
		message.To = destID
		err := n.SendMessage(message)
		if err != nil {
			fmt.Print(err)
			continue
		}
	}
}

//SendMessage sends a message to the desired recipient
func (n *Network) SendMessage(message Message) (err error) {
	if message.To == n.Myself.ID {
		n.RecieveChannel <- message
		return nil
	}
	messageByte, err := json.Marshal(message) 
	if check(err) {
		return err
	}
	remoteConn := n.Connections[message.To]
	if remoteConn == nil {
		fmt.Println(n.Connections)
		return fmt.Errorf("No connection to %v", message.To)
	}
	_, err = n.Connections[message.To].Write(messageByte)
	if check(err) {
		n.CloseConn(n.Connections[message.To])
		return err
	}
	return err

}

func (n *Network) printNetwork() {
	fmt.Printf("-- Connection table for node: %d--\n\n", n.Myself.ID)
	fmt.Printf("Node ID \t Local Address \t\t Remote address \n")
	for nodeid, TCPconn := range n.Connections {
		fmt.Printf("node %d\t%v\t %v\n", nodeid, TCPconn.LocalAddr(), TCPconn.RemoteAddr())
	}
	for i, TCPconn := range n.ClientConnections {
		fmt.Printf("Client %d\t%v\t %v\n", i, TCPconn.LocalAddr(), TCPconn.RemoteAddr())
	}
	fmt.Printf("\n --Connection table for node %d--\n", n.Myself.ID)

}
