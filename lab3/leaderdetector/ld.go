package leaderdetector

import "fmt"

// A MonLeaderDetector represents a Monarchical Eventual Leader Detector as
// described at page 53 in:
// Christian Cachin, Rachid Guerraoui, and Luís Rodrigues: "Introduction to
// Reliable and Secure Distributed Programming" Springer, 2nd edition, 2011.
type MonLeaderDetector struct { // TODO(student): Add needed fields
	suspected map[int]bool // suspected map
	currentLeader int
	nodeIDs   []int        // node ids for every node in cluster
	subscriber chan<- int
}

// NewMonLeaderDetector returns a new Monarchical Eventual Leader Detector
// given a list of node ids.
func NewMonLeaderDetector(nodeIDs []int) *MonLeaderDetector {
	// TODO(student): Add needed implementation
	suspected := make(map[int]bool)
	startingLeader := -1
   	for _, val  := range nodeIDs {
		   fmt.Println(val)
       if val > startingLeader {
        	startingLeader = val
        }
    }

	m := &MonLeaderDetector{suspected: suspected, currentLeader: startingLeader, nodeIDs: nodeIDs}
	return m
}

// Leader returns the current leader. Leader will return UnknownID if all nodes
// are suspected.
func (m *MonLeaderDetector) Leader() int {
	// TODO(student): Implement
	// Her må vi få tak i alle nodeID
	
	return m.currentLeader
	return UnknownID
}

// Suspect instructs the leader detector to consider the node with matching
// id as suspected. If the suspect indication result in a leader change
// the leader detector should publish this change to its subscribers.
func (m *MonLeaderDetector) Suspect(id int) {
	// TODO(student): Implement
	m.suspected[id] = true
	oldLeader := m.currentLeader 
	newLeader := m.changeLeader()
	if oldLeader != newLeader {
		//  A switch has been made, notify subscribers
		m.subscriber <- m.currentLeader
		//m.changeLeader()
	}
}

// Restore instructs the leader detector to consider the node with matching
// id as restored. If the restore indication result in a leader change
// the leader detector should publish this change to its subscribers.
func (m *MonLeaderDetector) Restore(id int) {
	// TODO(student): Implement
	m.suspected[id] = false
	oldLeader := m.currentLeader 
	newLeader := m.changeLeader()
	if oldLeader != newLeader {
		fmt.Println(newLeader)
		//  A switch has been made, notify subscribers
		m.subscriber <- m.currentLeader
		//for _, v := range(m.nodeIDs) {
		//	m.subscriber[v] <- m.currentLeader
		//}
		//m.changeLeader()
	}
}

// Subscribe returns a buffered channel which will be used by the leader
// detector to publish the id of the highest ranking node.
// The leader detector will publish UnknownID if all nodes become suspected.
// Subscribe will drop publications to slow subscribers.
// Note: Subscribe returns a unique channel to every subscriber;
// it is not meant to be shared.
func (m *MonLeaderDetector) Subscribe() <-chan int {
	// TODO(student): Implement
	subscriberline := make(chan int, 100)
	m.subscriber = subscriberline
	//subscriberline <- m.currentLeader
	//m.subscriber[int] = subscriberline
	//fmt.Println("Create subscriberline")
	return subscriberline
}

// TODO(student): Add other unexported functions or methods if needed.
func (m *MonLeaderDetector) changeLeader() int{
	max := -1

	for val := range m.nodeIDs {
		if val > max && !m.suspected[val]  {
        	max = val
        }
	}
	m.currentLeader = max
	return max
	
}