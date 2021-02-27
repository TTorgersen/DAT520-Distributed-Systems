// Leave an empty line above this comment.
package main

import (
	"fmt"
	"net"
	"strings"
)

// UDPServer implements the UDP Echo Server specification found at
// https://github.com/COURSE_TAG/assignments/tree/master/lab2/README.md#udp-echo-server
type UDPServer struct {
	conn    *net.UDPConn
	addr    *net.UDPAddr
	clients map[int]*net.UDPAddr
}

// NewUDPServer returns a new UDPServer listening on addr. It should return an
// error if there was any problem resolving or listening on the provided addr.
func NewUDPServer(addr string) (*UDPServer, error) {
	address, err := net.ResolveUDPAddr("udp", addr)
	if err != nil {
		return nil, err
	}

	conn, err := net.ListenUDP("udp", address)
	if err != nil {
		return nil, err
	}
	clients := make(map[int]*net.UDPAddr)
	return &UDPServer{conn, address, clients}, nil
}

// ServeUDP starts the UDP server's read loop. The server should read from its
// listening socket and handle incoming client requests as according to the
// the specification.
func (u *UDPServer) ServeUDP() {
	//defer u.conn.Close()
	fmt.Println("hello")
	buf := make([]byte, 512)
	for {
		// Read from socket
		fmt.Println("hello2")
		n, addr, err := u.conn.ReadFrom(buf)
		fmt.Println("hello3")
		if err != nil {
			println(err)
		}
		fmt.Println("hello4")
		fmt.Println(string(buf[0:n]))
		msg := string(buf[0:n])
		fmt.Println(msg)
		_, err = u.conn.WriteTo([]byte(msg), addr)
		if err != nil {
			println(err)
		}
	}
}

// socketIsClosed is a helper method to check if a listening socket has been
// closed.
func socketIsClosed(err error) bool {
	if strings.Contains(err.Error(), "use of closed network connection") {
		return true
	}
	return false
}

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
	send_address := local_address.IP.String() + ":" + "5000"
	//send_address := local_address.String()
	fmt.Println(send_address)
	server, err := NewUDPServer(send_address)
	if err != nil {
		fmt.Println(err)
	}
	server.ServeUDP()
}
