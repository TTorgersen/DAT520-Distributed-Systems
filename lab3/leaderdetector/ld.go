package leaderdetector

// A MonLeaderDetector represents a Monarchical Eventual Leader Detector as
// described at page 53 in:
// Christian Cachin, Rachid Guerraoui, and Luís Rodrigues: "Introduction to
// Reliable and Secure Distributed Programming" Springer, 2nd edition, 2011.
type MonLeaderDetector struct {
	Suspected     map[int]bool // suspected map
	CurrentLeader int
	nodeIDs       []int // node ids for every node in cluster
	subscribers   []chan int
}

// NewMonLeaderDetector returns a new Monarchical Eventual Leader Detector
// given a list of node ids.
func NewMonLeaderDetector(nodeIDs []int) *MonLeaderDetector {
	m := &MonLeaderDetector{
		Suspected:     make(map[int]bool),
		CurrentLeader: UnknownID,
		nodeIDs:       nodeIDs,
	}

	changed := m.ChangeLeader()
	if changed {
		m.NotifySubscribers()
	}

	return m
}

// Leader returns the current leader. Leader will return UnknownID if all nodes
// are suspected.
func (m *MonLeaderDetector) Leader() int {
	return m.CurrentLeader
}

// Suspect instructs the leader detector to consider the node with matching
// id as suspected. If the suspect indication result in a leader change
// the leader detector should publish this change to its subscribers.
func (m *MonLeaderDetector) Suspect(id int) {
	m.Suspected[id] = true
	changed := m.ChangeLeader()
	if changed {
		m.NotifySubscribers()
	}
}

// Restore instructs the leader detector to consider the node with matching
// id as restored. If the restore indication result in a leader change
// the leader detector should publish this change to its subscribers.
func (m *MonLeaderDetector) Restore(id int) {
	m.Suspected[id] = false
	changed := m.ChangeLeader()
	if changed {
		m.NotifySubscribers()
	}
}

// Subscribe returns a buffered channel which will be used by the leader
// detector to publish the id of the highest ranking node.
// The leader detector will publish UnknownID if all nodes become suspected.
// Subscribe will drop publications to slow subscribers.
// Note: Subscribe returns a unique channel to every subscriber;
// it is not meant to be shared.
func (m *MonLeaderDetector) Subscribe() <-chan int {
	ch := make(chan int, 3000)
	m.subscribers = append(m.subscribers, ch)
	return ch
}

// Change leader node
func (m *MonLeaderDetector) ChangeLeader() bool {
	leaderNode := UnknownID

	for _, node := range m.nodeIDs {
		if node > leaderNode && m.Suspected[node] == false {
			leaderNode = node
		}
	}
	if leaderNode != m.CurrentLeader {
		m.CurrentLeader = leaderNode
		return true
	}
	return false
}

func (m *MonLeaderDetector) NotifySubscribers() {
	for _, ch := range m.subscribers {
		ch <- m.CurrentLeader
	}
}
