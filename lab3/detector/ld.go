package detector

import "fmt"

// A MonLeaderDetector represents a Monarchical Eventual Leader Detector as
// described at page 53 in:
// Christian Cachin, Rachid Guerraoui, and Luís Rodrigues: "Introduction to
// Reliable and Secure Distributed Programming" Springer, 2nd edition, 2011.
type MonLeaderDetector struct {
	Suspected     map[int]bool // suspected map
	currentLeader int
	nodeIDs       []int // node ids for every node in cluster
	subscriber    []chan int
}

// NewMonLeaderDetector returns a new Monarchical Eventual Leader Detector
// given a list of node ids.
func NewMonLeaderDetector(nodeIDs []int) *MonLeaderDetector {
	suspected := make(map[int]bool)
	startingLeader := -1
	for _, val := range nodeIDs {
		fmt.Println(val)
		if val > startingLeader {
			startingLeader = val
		}
	}

	m := &MonLeaderDetector{Suspected: suspected, currentLeader: startingLeader, nodeIDs: nodeIDs}
	return m
}

// Leader returns the current leader. Leader will return UnknownID if all nodes
// are suspected.
func (m *MonLeaderDetector) Leader() int {

	return m.currentLeader
}

// Suspect instructs the leader detector to consider the node with matching
// id as suspected. If the suspect indication result in a leader change
// the leader detector should publish this change to its subscribers.
func (m *MonLeaderDetector) Suspect(id int) {

	m.Suspected[id] = true
	m.changeLeader()

}

// Restore instructs the leader detector to consider the node with matching
// id as restored. If the restore indication result in a leader change
// the leader detector should publish this change to its subscribers.
func (m *MonLeaderDetector) Restore(id int) {

	m.Suspected[id] = false
	m.changeLeader()

}

// Subscribe returns a buffered channel which will be used by the leader
// detector to publish the id of the highest ranking node.
// The leader detector will publish UnknownID if all nodes become suspected.
// Subscribe will drop publications to slow subscribers.
// Note: Subscribe returns a unique channel to every subscriber;
// it is not meant to be shared.
func (m *MonLeaderDetector) Subscribe() <-chan int {

	subscriberline := make(chan int, 100)
	m.subscriber = append(m.subscriber, subscriberline)
	return subscriberline
}

func (m *MonLeaderDetector) changeLeader() int {
	max := -1

	for val := range m.nodeIDs {
		if val > max && !m.Suspected[val] {
			max = val
		}
	}
	oldLeader := m.currentLeader
	m.currentLeader = max
	newLeader := max
	if oldLeader != newLeader {
		//  A switch has been made, notify subscribers
		m.notifySubscribers()
	}

	return max
}

func (m *MonLeaderDetector) notifySubscribers() int {
	for _, ch := range m.subscriber {
		ch <- m.currentLeader
	}
	return m.currentLeader
}
