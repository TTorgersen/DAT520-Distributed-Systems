package main

import (
	"bufio"
	mp "dat520/lab5/multipaxos"
	network "dat520/lab5/network"
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
	nrOfServers := 3
	delaymultiplier := 10

	thisClient := createRandomID()
	ip, port := GetOutboundIP()

	thisaddress := ip.String() + ":" + strconv.Itoa(port)
	tcpadd, err := net.ResolveTCPAddr("tcp", thisaddress)

	TCPListn, err := net.ListenTCP("tcp", tcpadd)

	fmt.Println("Hello R2-D2, your secret ID for this mission is:")
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
			if len(text) > 6 {
				if text[0:7] == "reconf " {
					nrOfNewServers := text[7:]
					fmt.Println("Reconfigure request received, new number of servers: ", nrOfNewServers)
					/* msg := new(network.Message)
					msg.Value.Command = text
					*/ //nrOfServers, _ = strconv.Atoi(nrOfNewServers)
					//recieveEntireMessage <- *msg

					//continue
				}
			}
			if text == "show seq" {
				fmt.Println("Current seq", seq)
				continue
			}
			if text == "show stats" {

				fmt.Println("---------------------------------------------------------------- ")
				fmt.Println("--------------------------This is the stats screen ------------- ")
				fmt.Println("Current seq", seq)
				fmt.Println("Number of servers configured ", nrOfServers)
				fmt.Println("nr of connections", len(connections))
				fmt.Println("Connections", connections)
				fmt.Println("Current delay", delay)
				fmt.Println("current delay multiplier", delaymultiplier)
				fmt.Println("---------------------------------------------------------------- ")
				continue
			}
			if text == "reset seq" {
				seq = 0
				fmt.Println("Current seq", seq)
				responseRecieved = true
				continue
			}
			if text == "help" {
				fmt.Println("---------------------------------------------------------------- ")
				fmt.Println("----------------------------This is the help screen------------- ")
				fmt.Println("---Available commands are:-------------------------------------- ")
				fmt.Println("-------------------------------show connections ---------------- ")
				fmt.Println("-------------------------------show stats ---------------------- ")
				fmt.Println("-------------------------------show seq ------------------------ ")
				fmt.Println("-------------------------------reset seq ----------------------- ")
				fmt.Println("-------------------------------help ---------------------------- ")
				fmt.Println("-------------------------------retry connections --------------- ")
				fmt.Println("-------------------------------reconf nrOfServers -------------- ")
				fmt.Println("---------------------------------------------------------------- ")
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
					fmt.Println("CODECHANGE, SENT ANYWAY DUE TO DEBUGGING")
					newVal := mp.Value{ClientID: thisClient, ClientSeq: seq,
						Command: text}
					CSend <- newVal
				}

			}

		}

	}()

	go func() {
		for {

			if len(connections) < nrOfServers-1 {
				fmt.Println(len(connections), nrOfServers)

				//go autoRetry(delay, netconf, connections, thisClient, recieveEntireMessage)

				timer1 := time.NewTimer(delay)
				<-timer1.C
				fmt.Println("Not all servers connected, trying to reconnect")
				connections = dialUp(netconf, connections, thisClient, recieveEntireMessage)
				if len(connections) < nrOfServers {
					fmt.Println("did not succeed to connect to all, doubling delay from ", delay, "to", delay*time.Duration(delaymultiplier))
					delay += delay * time.Duration(delaymultiplier)

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
			if msg.Type == "reconf" {
				sendNetworkMessage(msg, connections)
			}
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
	buffer := make([]byte, 2048, 2048)

	for {
		len, err := c.Read(buffer[0:])
		if err != nil {
			fmt.Print(err)
			key := findID(c, connections)
			fmt.Println("error detected, deleting connection to ", key)
			if key != -1 {
				delete(connections, key)

			}
			fmt.Print(err)
			return err
		}
		msg := network.Message{}
		err = json.Unmarshal(buffer[0:len], &msg)
		if err != nil {
			fmt.Println(err)
		}
		rchan <- msg
	}

}

func sendNetworkMessage(msg network.Message, connections map[int]net.Conn) (err error) {
	messageInBytes, err := json.Marshal(msg)
	if err != nil {
		fmt.Println(err)
		return err
	}
	i := 0
	for _, c := range connections {
		bytes, err := c.Write(messageInBytes)
		if err != nil {
			fmt.Println(err)
			return err
		}
		i++
		_ = bytes
	}
	fmt.Println("Network message of type", msg.Type, " was sent to ", i, " connections")
	return nil
}
func sendMessage(sendMsg mp.Value, connections map[int]net.Conn) (err error) {
	// changing new to other
	msg := network.Message{}
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
