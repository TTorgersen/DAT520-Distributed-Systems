package detector

// A MonLeaderDetector represents a Monarchical Eventual Leader Detector as
// described at page 53 in:
// Christian Cachin, Rachid Guerraoui, and Luís Rodrigues: "Introduction to
// Reliable and Secure Distributed Programming" Springer, 2nd edition, 2011.
type MonLeaderDetector struct {
	// TODO(student): Add needed fields
	nodes          []int        //Slice with all node IDs
	SuspectedNodes map[int]bool //Map with all node IDs and if they are suspected (true = suspected)
	leaderNode     int          //ID of the current leader node
	subscribers    []chan int   //slice with channels for all subribers. Needs a list so one can publish updates to each subscriber
}

// NewMonLeaderDetector returns a new Monarchical Eventual Leader Detector
// given a list of node ids.
func NewMonLeaderDetector(nodeIDs []int) *MonLeaderDetector {
	m := &MonLeaderDetector{
		nodes:          nodeIDs, // What if the there is no values (nil) in the nodeIDs slice?
		SuspectedNodes: make(map[int]bool),
		leaderNode:     UnknownID, // Sets the leaderNode entry to "UnkownID" which is a constant int = -1 from defs.go
	}

	changed := m.LeaderChange()
	if changed {
		m.Publish()
	}

	return m
}

// Leader returns the current leader. Leader will return UnknownID if all nodes
// are suspected.
func (m *MonLeaderDetector) Leader() int {
	// TODO(student): Implement
	return m.leaderNode
}

// Suspect instructs m to consider the node with matching id as suspected. If
// the suspect indication result in a leader change the leader detector should
// this publish this change its subscribers.
func (m *MonLeaderDetector) Suspect(id int) {
	// TODO(student): Implement
	m.SuspectedNodes[id] = true
	//if id == m.leaderNode {
	changed := m.LeaderChange()
	if changed {
		m.Publish()
	}
	//}
}

// Restore instructs m to consider the node with matching id as restored. If
// the restore indication result in a leader change the leader detector should
// this publish this change its subscribers.
func (m *MonLeaderDetector) Restore(id int) {
	// TODO(student): Implement
	m.SuspectedNodes[id] = false
	//if id >= m.leaderNode {
	changed := m.LeaderChange()
	if changed {
		m.Publish()
	}
	//}
}

// Subscribe returns a buffered channel that m on leader change will use to
// publish the id of the highest ranking node. The leader detector will publish
// UnknownID if all nodes become suspected. Subscribe will drop publications to
// slow subscribers. Note: Subscribe returns a unique channel to every
// subscriber; it is not meant to be shared.
func (m *MonLeaderDetector) Subscribe() <-chan int {
	// TODO(student): Implement
	c := make(chan int, 3000)                //makes a channel for subscribing process
	m.subscribers = append(m.subscribers, c) //Appends the newly creates channel to the list of all subscribers (with their channels)
	return c                                 //returns the channel to the subscriber

}

// LeaderChange is used to change leaderNode in the struct MonLeaderDetector
func (m *MonLeaderDetector) LeaderChange() bool {
	ln := UnknownID //Reset leaderNode ID
	for _, node := range m.nodes {
		if node > ln && m.SuspectedNodes[node] == false { //If node is bigger than ln and not suspected...
			ln = node //set ln = node id
		}
	}
	if ln != m.leaderNode {
		m.leaderNode = ln
		return true
	}
	return false
}

// Publish the new nodeLeader to all subscribers in m.subscribers (over "dedicated" channels)
func (m *MonLeaderDetector) Publish() { //look into creating goroutines?
	for _, c := range m.subscribers {
		c <- m.leaderNode
	}
}
