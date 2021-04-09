package network

import (
	fd "dat520/lab3/failuredetector"
	mp "dat520/lab5/multipaxos"
	"encoding/json"
	"fmt"
	"os"
	//"log"
	"net"
	//"strconv"
	"strings"
	"sync"
)

//Netconf Network config struct
type Netconf struct {
	Myself int
	Nodes  []Node
}

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
	ClientIP  string       //client infor
	Alive		bool  //should node be alive or not
	Heartbeat fd.Heartbeat //heartbeat msg
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
	buffer := make([]byte, 1024, 1024)
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
			fmt.Println("Closing connection from",message.To)
			connection.Close()
			delete(n.Connections, message.To)
		}
	}
	if message.Type == "Response" {
		if connection, ok := n.ClientConnections[message.ClientIP]; ok{
		_, err = connection.Write(messageByte)
		if err != nil {
			connection.Close()
			delete(n.ClientConnections, message.ClientIP)
		}
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
/* 
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
			//CHANGE 7
			fmt.Println("TRYING TO SEND TCPDIAL", TCPDial)
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



func (n *Network) ListenForConnection(TCPConnection *net.TCPConn) (err error) {

    // defers closing connection until the end
    defer n.CloseConn(TCPConnection)

    //etarnal for loop to handle listening to connections
    for {
        var buffer [2048]byte
        len, err := TCPConnection.Read(buffer[:])
		if len > 1000{
			fmt.Println("A clash has occured, skipping message, length: ", len)
			continue
		}
        if err != nil {
          if strings.Contains(err.Error(), "use of closed network connection") {
            break
          }
          continue
        } // CHANGE FROM message := new(Message) to message:= Message{}
        message := Message{}
		//fmt.Println("message len: ", len)
        err = json.Unmarshal(buffer[:len], &message)
        if check(err) {
            return err
        }
        
        n.RecieveChannel <- message
	}
	return nil
}
//ListenForConnection shall be initialized in a separate go routine
// in order
/* func (n *Network) ListenForConnection(TCPConnection *net.TCPConn) (err error) {

	// defers closing connection until the end
	defer n.CloseConn(TCPConnection)

	buffer := make([]byte, 2048, 2048)

	//etarnal for loop to handle listening to connections
	for { // CHANGE 8
		len, err := TCPConnection.Read(buffer)
		if err != nil{
			continue
		}
		//fmt.Println("BUFFER 0, ", string(buffer))
		message := new(Message)
		err = json.Unmarshal(buffer[:len], &message)
		if check(err) { // CHANGE 5 HERE IT STRUGGLES
			fmt.Println("STRUGGLING TO UNMARSHAL MESSAGE, TYPE", message.Type, " and message: |", message, "| MESSAGE STOP")
			//CHANGE 9 THIS KEEPS THE CONNECTION IN PLACE ERR DOES NOT
			continue
			//return err
		}
		// CHANGE 6 
		if message.Type != "Heartbeat"{
		fmt.Println("MESSAGE SUCCESSFULLY RECIEVED, TYPE: ", message.Type)
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
	//fmt.Println("the nodes in remote", n.Nodes)
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
			fmt.Println("TRYING TO TCPACCEPT TOR CHANGE ", TCPaccept)
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
					messageByte, err := json.Marshal(message)
						if err != nil {
							log.Print(err)
							continue
						}
						// IKKE DENNE WRITE MEST SANNSYNLIG da vi ville fått problem på client
						_, err = n.Connections[7].Write(messageByte)
						if err != nil {
							log.Print(err)
						}
						//_, err = n.Connections[8].Write(messageByte)
						//if err != nil {
						//	log.Print(err)
						//}
					/* for _, conns := range n.ClientConnections {
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
		// CHANGE 3
		fmt.Println("STRUGGLING TO MARSHALL MESSAGE", message)
		return err
	}
	remoteConn := n.Connections[message.To]
	if remoteConn == nil {
		fmt.Println(n.Connections)
		return fmt.Errorf("No connection to %v", message.To)
	}
	_, err = n.Connections[message.To].Write(messageByte)
	if check(err) { // CHANGE 4
		fmt.Println("STRUGGLING TO WRITE MESSAGE", message, " Messagebyte: ", messageByte)
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
 */