package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net"
	"os"
)

//Nodes ...
type Nodes struct {
	Self int `json:"self"`	
	Nodes []Node `json:"nodes"`
}

//Node ...
type Node struct {
	ID   int    `json:"id"`
	IP   string `json:"ip"`
	Port string `json:"port"`
}

/* func handleConnection(conn net.Conn) {
	defer conn.Close()
	scanner := bufio.NewScanner(conn)
	for scanner.Scan() {
		message := scanner.Text()
		fmt.Println("Message Received:", message)
		newMessage := strings.ToUpper(message)
		conn.Write([]byte(newMessage + "\n"))
	}

	if err := scanner.Err(); err != nil {
		fmt.Println("error:", err)
	}
}
*/
//GetOutboundIP ...
/* func GetOutboundIP() (net.IP, int) {
	conn, err := net.Dial("tcp", "127.0.0.1:8100")
	if err != nil {
		fmt.Println(err)
	}
	defer conn.Close()

	localAddr := conn.LocalAddr().(*net.TCPAddr)
	fmt.Println(localAddr)

	return localAddr.IP, localAddr.Port
}
*/
func main() {
	/* 	arguments := os.Args
	   	if len(arguments) == 1 {
	   		fmt.Println("Provide port number")
	   		return
	   	} */
	/*
		PORT := ":" + arguments[1]
		ln, err := net.Listen("tcp", "127.0.0.1"+PORT)
		if err != nil {
			log.Fatal(err)
		}
		fmt.Println("Accept connection on port") */

	//Connect to another server
	/* 	PORT2 := ":" + arguments[1]
	   	fmt.Println("Sending to port", PORT2) */
	jsonFile, _ := os.Open("netconf.json")
	defer jsonFile.Close()

	byteValue, _ := ioutil.ReadAll(jsonFile)


	var nodes Nodes
	json.Unmarshal(byteValue, &nodes)

	connection := ""
 	for i := 0; i < len(nodes.Nodes); i++ {
		if nodes.Self == nodes.Nodes[i].ID {
			connection = nodes.Nodes[i].IP +":"+ nodes.Nodes[i].Port
		}
	} 
	fmt.Println(connection)

	conn, _ := net.Dial("tcp", connection)
	conn.Write([]byte("Message from client B"))

	for {
		message, err := bufio.NewReader(conn).ReadString('\n')
		print(message)
		if err != nil {
			fmt.Println(err)
			return
		}
		fmt.Println(message)
	}

	/* 	if err != nil {
	   		panic(err.Error())
	   	}
	   	fmt.Fprintf(conn, "From client 2\n")
	   	message, _ := bufio.NewReader(conn).ReadString('\n')
	   	fmt.Print(message)
	   	conn.Close()

	   	for {
	   		conn, err := ln.Accept()
	   		if err != nil {
	   			log.Fatal(err)
	   		}
	   		fmt.Println("Calling handleConnection")
	   		go handleConnection(conn)
	   	} */

}
