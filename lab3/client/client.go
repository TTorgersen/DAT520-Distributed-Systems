package main

import (
	"bufio"
	"fmt"
	"net"
	"strings"
)

func handleConnection(conn net.Conn) {
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
	conn, _ := net.Dial("tcp", "127.0.0.1:8100")

	conn.Write([]byte("Message from client A"))

	for {
		message, err := bufio.NewReader(conn).ReadString('\n')
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
