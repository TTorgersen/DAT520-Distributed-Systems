package main

import (
	failure_detector "dat520/lab3/failuredetector"
	leader_detector "dat520/lab3/leaderdetector"
	"net"
)

var address = []string{"localhost:12111", "localhost:12112", "localhost:12113"}

type Message struct {
	Command   string
	Parameter interface{}
}

type netAddress struct {
	IP   string
	Port string
}

type DistributedNetwork struct {
	connections  map[int]net.UDPAddr
	eventLeader  leader_detector.MonLeaderDetector
	eventFailure failure_detector.FailureDetector
}

func NewDistributedNetwork() {

}
