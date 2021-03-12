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

func main() {
	fmt.Println("Hello R2-D2, your secret ID for this mission is:")

	thisClient := createRandomID()
	ip, port := GetOutboundIP()

	thisaddress := ip.String() + ":" + strconv.Itoa(port)
	tcpadd, err := net.ResolveTCPAddr("tcp", thisaddress)

	TCPListn, err := net.ListenTCP("tcp", tcpadd)

	fmt.Println(thisClient, " and listen", TCPListn)
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

	// send channel value
	CSend := make(chan mp.Value, 2000000)

	// recieve channel network message
	recieveEntireMessage := make(chan network.Message, 2000000)

	// we need a map to store connection info
	//Setting up connections
	connections := make(map[int]net.Conn)
	connections = dialUp(netconf, connections, thisClient, recieveEntireMessage)
	delay := 3 * time.Second
	// sequence number
	seq := 0
	responseRecieved := true

	go func() {
		for {
			reader := bufio.NewReader(os.Stdin)

			fmt.Print("--> ")
			text, _ := reader.ReadString('\n')

			text = text[:len(text)-1]
			if text == "retry connections" {
				connections = dialUp(netconf, connections, thisClient, recieveEntireMessage)
				continue
			}
			if text == "show seq" {
				fmt.Println("Current seq", seq)
				continue
			}
			if text == "reset seq" {
				seq = 0
				fmt.Println("Current seq", seq)
				responseRecieved = true
				continue
			}
			if text == "show connections" {
				fmt.Println(connections)
			} else {
				if responseRecieved {
					// NEED TO SEND data to servers
					newVal := mp.Value{ClientID: thisClient, ClientSeq: seq,
						Command: text}
					CSend <- newVal

					responseRecieved = false
				} else {
					fmt.Println("Command can not be entered as no response to previous command has arrived")
				}

			}

		}

	}()

	go func() {
		for {

			if len(connections) < 3 {

				//go autoRetry(delay, netconf, connections, thisClient, recieveEntireMessage)

				timer1 := time.NewTimer(delay)
				<-timer1.C
				fmt.Println("Not all servers connected, trying to reconnect")
				connections = dialUp(netconf, connections, thisClient, recieveEntireMessage)
				if len(connections) < 3 {
					fmt.Println("did not succeed to connect to all, doubling delay from ", delay, "to", delay*2)
					delay += delay

				}
			}
		}
	}()

	for {

		select {
		case sendMsg := <-CSend:
			sendMessage(sendMsg, connections)
			// HERE WE SEND TO SERVER
		case msg := <-recieveEntireMessage:
			//HERE WE RECIEVE
			if msg.Type == "Response" {
				//fmt.Println(msg)
				if msg.Response.ClientSeq == seq {
					responseRecieved = true
					seq++
				}
				fmt.Println("melding fra serveren", msg.Response)
			}
		}

	}
}

func autoRetry(delay time.Duration, netconf network.Netconf, connections map[int]net.Conn, thisClient string, recieveEntireMessage chan network.Message) map[int]net.Conn {
	timer1 := time.NewTimer(delay)
	<-timer1.C
	fmt.Println("Not all servers connected, trying to reconnect")
	connections = dialUp(netconf, connections, thisClient, recieveEntireMessage)
	return connections
}

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

func dialUp(netconf network.Netconf, connections map[int]net.Conn, thisClient string, recieveEntireMessage chan network.Message) map[int]net.Conn {

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
		go listenForConnectiontest(tcpConn, recieveEntireMessage, connections)
	}

	if len(connections) == 0 {
		fmt.Println("No connections detected, you are all alone, write \"retry connections\" to try again.")
	} else {
		fmt.Println(len(connections), " Connections detected")
		//ip, port := GetOutboundIP()
		//str := "INIT|" + ip.String() + ":" + strconv.Itoa(port)
		//init := mp.Value{ClientID: thisClient, Command: str}
		//sendMessage(init, connections)
	}
	return connections
}

func listenForConnectiontest(c net.Conn, rchan chan network.Message, connections map[int]net.Conn) (err error) {
	defer c.Close()
	buffer := make([]byte, 1024, 1024)

	for {
		len, err := c.Read(buffer[0:])
		if err != nil {
			fmt.Print(err)
			key := findID(c, connections)
			fmt.Println("error detected, deleting connection to ", key)
			if key != -1 {
				delete(connections, key)

			}
			return err
			fmt.Print(err)
		}
		msg := new(network.Message)
		err = json.Unmarshal(buffer[0:len], &msg)
		if err != nil {
			fmt.Println(err)
		}
		rchan <- *msg
	}

}
func sendMessage(sendMsg mp.Value, connections map[int]net.Conn) (err error) {
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

func findID(tcpConn net.Conn, connections map[int]net.Conn) (key int) {
	for mapKey, conn := range connections {
		if tcpConn == conn {
			return mapKey
		}
	}

	return -1
}
