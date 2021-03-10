package main

import (
	"bufio"
	network "dat520/LAB3FINAL/network"
	mp "dat520/lab4/multipaxos"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"math/rand"
	"net"
	"os"
	"strconv"
	"strings"
	"time"
)

// Get preferred outbound ip of this machine "Copied from online"
func GetOutboundIP() net.IP {
	conn, err := net.Dial("udp", "8.8.8.8:80")
	if err != nil {
		log.Fatal(err)
	}
	defer conn.Close()

	localAddr := conn.LocalAddr().(*net.UDPAddr)

	return localAddr.IP
}

// create a random id from rand + ip. (uneccesary complicated)
func createRandomID() string {
	ip := GetOutboundIP().String()
	//ipint, _ := strconv.Atoi(ip)
	ID, _ := strconv.Atoi(strings.Join(strings.Split(ip, "."), ""))
	rand.Seed(time.Now().UnixNano())
	return strconv.Itoa((ID * rand.Intn(100)) % 1000 + 2	)
}

func main() {
	fmt.Println("Hello R2-D2, your secret ID for this mission is:")
	thisClient := createRandomID()
	fmt.Println(thisClient)
	// get configuration file
	netconfigureFile, err := os.Open("app/netconf.json")
	if err != nil {
		log.Print(err)
		return
	}
	defer netconfigureFile.Close()

	//1.2 Read the network config file as byte array
	byteVal, _ := ioutil.ReadAll(netconfigureFile)

	//initialize a netconfig struct to save the netconfig
	var netconf network.Netconf
	err = json.Unmarshal(byteVal, &netconf)
	if err != nil {
		log.Print(err)
	}

	//1.3 Now netconf has the info from json file, step 1 complete
	CSend := make(chan mp.Value, 100)
	Crecieve := make(chan mp.Value, 100)
	// we need a map to store connection info
	connections := make(map[int]*net.TCPConn)
	for _, e := range netconf.Nodes {
		fmt.Println("Testin", e.ID)
		rAddr, err := net.ResolveTCPAddr("tcp", e.IP+":"+ strconv.Itoa(e.Port))
		if err != nil {
			fmt.Println(err)
			return
		}
		fmt.Println("Raddr resolved", rAddr)
		tcpConn, err := net.DialTCP("tcp", nil, rAddr)
		if err != nil {
			fmt.Println(err)
			return
		}
		// NOE SKJER
		fmt.Println("connection connected", tcpConn)
		connections[e.ID] = tcpConn
		go listenForConnection(tcpConn, Crecieve)
	}


	go func() {
		for {
			reader := bufio.NewReader(os.Stdin)

			fmt.Print("--> ")
			text, _ := reader.ReadString('\n')

			// NEED TO SEND data to servers
			newVal := mp.Value{ClientID: thisClient,
				Command: text}
			CSend <- newVal

		}

	}()

	for {
		select {
		case sendMsg := <-CSend:
			// HERE WE SEND TO SERVER
			// dummy function
			sendMessage(sendMsg, connections)

			// fake send:

			//fmt.Println("Sent to server: ", sendMsg)
		case recieve := <-Crecieve:
			// here we recieve values.
			test := recieve.Command + " Executed"
			fmt.Println(test)
		}

	}
}
func sendMessage(sendMsg mp.Value, connections map[int]*net.TCPConn) (err error ) {
	//channel chan<- mp.Value
	//channel <- sendMsg
	msg := new(network.Message)
	msg.Type = "Value"
	msg.Value = sendMsg
	messageInBytes, err := json.Marshal(msg)
	if err != nil{
		fmt.Println(err)
		return err
	}
	for _, c := range connections{
		bytes, err := c.Write(messageInBytes)
		if err != nil{
			fmt.Println(err)
			return err 
		}
		fmt.Println("message of n bytes sent: ", bytes)
	} 
	
	return nil
}

func listenForConnection(TCPConnection *net.TCPConn, recieveChan chan<- mp.Value) (err error) {

	// defers closing connection until the end
	defer TCPConnection.Close()

	buffer := make([]byte, 1024, 1024)

	//etarnal for loop to handle listening to connections
	for {
		len, _ := TCPConnection.Read(buffer[0:])
		message := new(network.Message)
		err = json.Unmarshal(buffer[0:len], &message)
		if err != nil {
			fmt.Print(err)
			continue	 

		}
		if message.Type == "Value"{
			recieveChan <- message.Value
		}
		

	}
}
