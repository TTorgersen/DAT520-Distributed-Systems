//package network

import (
	fd "dat520/lab3/failuredetector"
	mp "dat520/lab4/multipaxos"
	"encoding/json"
	"fmt"
	"net"
	"os"
	"strings"
	"sync"
)

// Struct for a Node in the network
type Node struct {
	ID        int
	IP        string
	Port      int
	TCPaddr   *net.TCPAddr
	TCPListen *net.TCPListener
}

// struct for the network with server connections
type Network struct {
	Myself            Node
	Nodes             []Node
	Connections       map[int]*net.TCPConn
	ClientConnections map[string]*net.TCPConn
	RecieveChannel    chan Message
	//SendChannel    chan Message
}
type Message struct {
	Type      string //what kind of message is it
	From      int
	To        int
	ClientIP  string       //client infor
	Heartbeat fd.Heartbeat //heartbeat msg
	Accept    mp.Accept    //accept msg
	Promise   mp.Promise   //promise msg
	Prepare   mp.Prepare   //prepare msg
	Learn     mp.Learn     //learn msg
	Value     mp.Value     //value msg
	Response  mp.Response  //response msg
}

var Mutex = &sync.Mutex{}

// Initiace the Network
func InitializeNetwork(nodes []Node, myself Node) (network Network) {
	reciveChann := make(chan Message, 2000000)
	//sendChann := make(chan Message, 2000000)
	network = Network{
		Nodes:             []Node{},
		Connections:       map[int]*net.TCPConn{},
		ClientConnections: map[string]*net.TCPConn{},
		RecieveChannel:    reciveChann,
		//SendChannel:    sendChann,
	}
	network.Myself = myself
	for _, node := range nodes {
		network.Nodes = append(network.Nodes, node)
	}
	return network
}

// Start the server
func (n *Network) StartServer() (err error) {
	// Make a listener for self
	TCPListen, err := net.ListenTCP("tcp", n.Myself.TCPaddr)
	if err != nil {
		fmt.Print(err)
	}
	go func() {
		defer TCPListen.Close()
		for {
			// Accept incomming dials
			TCPAccept, err := TCPListen.AcceptTCP()
			if err != nil {
				fmt.Print(err)
			}
			// Find the remote address of the incomming dial
			remoteAddr := TCPAccept.RemoteAddr().String()
			ip := strings.Split(remoteAddr, ":")[0]
			clientNode := true
			// If the IP is listed in the network, add the connection in the network
			for _, node := range n.Nodes {
				if node.IP == ip {
					Mutex.Lock()
					n.Connections[node.ID] = TCPAccept
					fmt.Println("Accepted connection from:" + fmt.Sprint(node.ID))
					Mutex.Unlock()
					clientNode = false
					// Listen on the connection if it is part of the network
				}
			}
			if clientNode {
				n.ClientConnections[ip] = TCPAccept
				fmt.Println("accepted connection from client: " + ip)
				fmt.Println(n.ClientConnections)
			}
			go n.Listen(TCPAccept)
		}
	}()
	return err
}

func (n *Network) Listen(conn *net.TCPConn) {
	buffer := make([]byte, 512, 512)
	for {
		len, err := conn.Read(buffer[0:])
		if err != nil {
			continue
		}
		message := Message{}
		json.Unmarshal(buffer[:len], &message)
		n.RecieveChannel <- message
	}
}

func (n *Network) Dial() {
	for _, node := range n.Nodes {
		conn, err := net.DialTCP("tcp", nil, node.TCPaddr)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Fatal error: %s\n", err)
			continue
		}
		n.Connections[node.ID] = conn
		go n.Listen(conn)
	}
}

func (n *Network) SendMessage(message Message) {
	messageByte, err := json.Marshal(message)
	if err != nil {
		fmt.Println(err)
	}
	if message.To == n.Myself.ID {
		n.RecieveChannel <- message
	} else if connection, ok := n.Connections[message.To]; ok {
		_, err = connection.Write(messageByte)
		if err != nil {
			connection.Close()
			delete(n.Connections, message.To)
		}
	}
	if message.Type == "Response" {
		connection := n.ClientConnections[message.ClientIP]
		_, err = connection.Write(messageByte)
		if err != nil {
			connection.Close()
			delete(n.ClientConnections, message.ClientIP)
		}
	}
}

//SendCommand to other modules
func (n *Network) SendMessageBroadcast(message Message, destination []int) {
	for _, destID := range destination {
		message.To = destID
		n.SendMessage(message)
	}
}