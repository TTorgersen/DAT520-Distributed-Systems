package main

import (
	"bufio"
	mp "dat520/lab4/multipaxos"
	network "dat520/lab4/network"
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
func GetOutboundIP() (net.IP, int) {
	conn, err := net.Dial("udp", "8.8.8.8:80")
	if err != nil {
		log.Fatal(err)
	}
	defer conn.Close()

	localAddr := conn.LocalAddr().(*net.UDPAddr)

	return localAddr.IP, localAddr.Port
}

// create a random id from rand + ip. (uneccesary complicated)
func createRandomID() string {
	ip, _ := GetOutboundIP()
	str := ip.String()
	//ipint, _ := strconv.Atoi(ip)
	ID, _ := strconv.Atoi(strings.Join(strings.Split(str, "."), ""))
	rand.Seed(time.Now().UnixNano())
	return strconv.Itoa((ID*rand.Intn(100))%1000 + 2)
}

func main() {
	fmt.Println("Hello R2-D2, your secret ID for this mission is:")
	thisClient := createRandomID()
	fmt.Println(thisClient)
	// get configuration file
	netconfigureFile, err := os.Open("../app/netconf.json")
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
	needInit := false

	connections := make(map[int]net.Conn)

	for _, e := range netconf.Nodes {
		fmt.Println("Testin", e.ID)

		rAddr, err := net.ResolveTCPAddr("tcp", e.IP+":"+strconv.Itoa(e.Port))
		if err != nil {
			fmt.Println(err)
			return
		}
		fmt.Println("Raddr resolved", rAddr)
		stringAddr := rAddr.String()
		d := net.Dialer{Timeout: 3 * time.Second}
		tcpConn, err := d.Dial("tcp", stringAddr)
		//tcpConn, err := net.DialTCP("tcp", nil, rAddr)

		if err != nil {
			fmt.Println(err)
			continue
		}
		// NOE SKJER
		fmt.Println("connection connected", tcpConn)
		needInit = true
		connections[e.ID] = tcpConn
		go listenForConnection(tcpConn, Crecieve, connections)
	}

	if len(connections) == 0 {
		fmt.Println("No connections detected, you are all alone, write \"retry connections\" to try again. or \"show connections\" to show them")
	} else {
		fmt.Println(len(connections), " Connections detected")
	}

	go func() {
		for {
			reader := bufio.NewReader(os.Stdin)

			fmt.Print("--> ")
			text, _ := reader.ReadString('\n')

			text = text[:len(text)-1]
			if text == "retry connections" {
				connections = dialUp(netconf, connections, Crecieve, needInit, thisClient)
				continue
			}
			if text == "show connections" {
				fmt.Println(connections)
			} else {
				// NEED TO SEND data to servers
				newVal := mp.Value{ClientID: thisClient,
					Command: text}
				CSend <- newVal
			}

		}

	}()

	for {
		if needInit == true {
			ip, port := GetOutboundIP()
			str := "INIT|" + ip.String() + ":" + strconv.Itoa(port)
			init := mp.Value{ClientID: thisClient, Command: str}
			sendMessage(init, connections)
			needInit = false
		}
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

func dialUp(netconf network.Netconf, connections map[int]net.Conn, Crecieve chan mp.Value, needinit bool, thisClient string) map[int]net.Conn {

	for _, e := range netconf.Nodes {
		fmt.Println("Testin", e.ID)

		rAddr, err := net.ResolveTCPAddr("tcp", e.IP+":"+strconv.Itoa(e.Port))
		if err != nil {
			fmt.Println(err)
			return nil
		}
		fmt.Println("Raddr resolved", rAddr)
		stringAddr := rAddr.String()
		d := net.Dialer{Timeout: 3 * time.Second}
		tcpConn, err := d.Dial("tcp", stringAddr)
		//tcpConn, err := net.DialTCP("tcp", nil, rAddr)

		if err != nil {
			fmt.Println(err)
			continue
		}
		// NOE SKJER

		fmt.Println("connection connected", tcpConn)
		connections[e.ID] = tcpConn
		go listenForConnection(tcpConn, Crecieve, connections)
	}

	if len(connections) == 0 {
		fmt.Println("No connections detected, you are all alone, write \"retry connections\" to try again.")
	} else {
		fmt.Println(len(connections), " Connections detected")
		ip, port := GetOutboundIP()
		str := "INIT|" + ip.String() + ":" + strconv.Itoa(port)
		init := mp.Value{ClientID: thisClient, Command: str}
		sendMessage(init, connections)
	}
	return connections
}

func sendMessage(sendMsg mp.Value, connections map[int]net.Conn) (err error) {
	//channel chan<- mp.Value
	//channel <- sendMsg
	msg := new(network.Message)
	msg.Type = "Value"
	msg.Value = sendMsg
	messageInBytes, err := json.Marshal(msg)
	if err != nil {
		fmt.Println(err)
		return err
	}
	for _, c := range connections {
		bytes, err := c.Write(messageInBytes)
		if err != nil {
			fmt.Println(err)
			return err
		}
		fmt.Println("message of n bytes sent: ", bytes)
	}

	return nil
}

func listenForConnection(TCPConnection net.Conn, recieveChan chan<- mp.Value, connections map[int]net.Conn) (err error) {

	// defers closing connection until the end
	defer TCPConnection.Close()
	//TCPConnection.SetDeadline(time.Now().Add(3*time.Second))
	buffer := make([]byte, 1024, 1024)

	//etarnal for loop to handle listening to connections
	for {
		len, err := TCPConnection.Read(buffer[0:])
		if err != nil {
			fmt.Print(err)
			key := findID(TCPConnection, connections)
			fmt.Println("error detected, deleting connection to ", key)
			if key != -1 {
				delete(connections, key)
			}
			return err
		}
		message := new(network.Message)
		err = json.Unmarshal(buffer[0:len], &message)
		if err != nil {
			//fmt.Print(err)
			//key := findID(TCPConnection, connections)
			//fmt.Println("error detected, deleting connection to ", key)
			//delete(connections, key)
			continue
			return err

		}
		if message.Type == "Value" {
			recieveChan <- message.Value
		}

	}
}

func findID(tcpConn net.Conn, connections map[int]net.Conn) (key int) {
	for mapKey, conn := range connections {
		if tcpConn == conn {
			return mapKey
		}
	}

	return -1
}
