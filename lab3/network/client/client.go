package main

import (
	"fmt"
	"net"
)

type UDPClient struct {
	conn *net.UDPConn
	addr *net.UDPAddr
}

func NewUDPClient(addr string) (*UDPClient, error) {
	address, err := net.ResolveUDPAddr("udp", addr)
	if err != nil {
		return nil, err
	}
	conn, err := net.DialUDP("udp", nil, address)
	if err != nil {
		return nil, err
	}
	return &UDPClient{conn, address}, nil
}

// SendCommand sends the command cmd with payload txt as a UDP packet to
// address updAddr. SendCommand prints errors to output.
//

func localAddress() *net.UDPAddr {
	conn, error := net.Dial("udp", "8.8.8.8:80")
	if error != nil {
		fmt.Println(error)
	}
	defer conn.Close()
	ipAddress := conn.LocalAddr().(*net.UDPAddr)
	return ipAddress
}

func main() {
	local_address := localAddress()
	send_address := string(local_address.IP) + ":" + "5000"
	client, err := NewUDPClient(send_address)
	if err != nil {
		fmt.Println(err)
	}
	var buf [512]byte
	_, err = client.conn.Write([]byte(send_address))
	if err != nil {
		fmt.Println(err)
	}
	n, err := client.conn.Read(buf[0:])
	if err != nil {
		fmt.Println(err)
	}
	fmt.Println(string(buf[0:n]))
}
