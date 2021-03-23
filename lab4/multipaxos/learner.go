package multipaxos


// Learner represents a learner as defined by the Multi-Paxos algorithm.
type Learner struct { // TODO(student): algorithm and distributed implementation
	// Add needed fields
	id         int
	nrOfNodes  int
	round      Round
	lrnSlots   map[SlotID][]Learn
	lrnSent    map[SlotID]bool
	decidedOut chan<- DecidedValue
	lrnValues  chan Learn
	stop       chan struct{}
}

// NewLearner returns a new Multi-Paxos learner. It takes the
// following arguments:
//
// id: The id of the node running this instance of a Paxos learner.
//
// nrOfNodes: The total number of Paxos nodes.
//
// decidedOut: A send only channel used to send values that has been learned,
// i.e. decided by the Paxos nodes.
func NewLearner(id int, nrOfNodes int, decidedOut chan<- DecidedValue) *Learner {
	// TODO(student): algorithm and distributed implementation
	newLearner := &Learner{id: id, nrOfNodes: nrOfNodes, round: NoRound, lrnSlots: make(map[SlotID][]Learn), lrnSent: make(map[SlotID]bool), decidedOut: decidedOut, lrnValues: make(chan Learn, 42000), stop: make(chan struct{})}
	return newLearner
}

// Start starts l's main run loop as a separate goroutine. The main run loop
// handles incoming learn messages.
func (l *Learner) Start() {
	go func() {
		for {
			// TODO(student): distributed implementation
			select {
			case lrn := <-l.lrnValues:
				if val, slot, ok := l.handleLearn(lrn); ok == true {
					l.decidedOut <- DecidedValue{SlotID: slot, Value: val}
				}
			case <-l.stop:
				break
			}
		}
	}()
}

// Stop stops l's main run loop.
func (l *Learner) Stop() {
	// TODO(student): distributed implementation
	l.stop <- struct{}{}
}

// DeliverLearn delivers learn lrn to learner l.
func (l *Learner) DeliverLearn(lrn Learn) {
	// TODO(student): distributed implementation
	l.lrnValues <- lrn
}

// Internal: handleLearn processes learn lrn according to the Multi-Paxos
// algorithm. If handling the learn results in learner l emitting a
// corresponding decided value, then output will be true, sid the id for the
// slot that was decided and val contain the decided value. If handleLearn
// returns false as output, then val and sid will have their zero value.
func (l *Learner) handleLearn(learn Learn) (val Value, sid SlotID, output bool) {
	// TODO(student): algorithm implementation
	majority := (l.nrOfNodes / 2) + 1

	if learn.Rnd < l.round { //check if current round is less than learned round
		return val, sid, false
	} else if learn.Rnd == l.round {
		if learnsInSlot, ok := l.lrnSlots[learn.Slot]; ok {
			for _, learnsInSlot := range learnsInSlot { //check if we have already learned this info
				if learnsInSlot.From == learn.From && learnsInSlot.Rnd == learn.Rnd { //if we have learned from this node from this round, ignore it
					return val, sid, false
				}
			}
			l.lrnSlots[learn.Slot] = append(l.lrnSlots[learn.Slot], learn)

		} else {
			l.lrnSlots[learn.Slot] = append(l.lrnSlots[learn.Slot], learn)
		}
	} else { //Update l.round if learn.rnd is larger
		l.round = learn.Rnd
		l.lrnSlots[learn.Slot] = nil
		l.lrnSlots[learn.Slot] = append(l.lrnSlots[learn.Slot], learn)
	}

	if l.lrnSent[learn.Slot] { //check if quorum is reached for this slotID
		return val, sid, false
	}

	if len(l.lrnSlots[learn.Slot]) >= majority { //check if quroum larger than majority
		l.lrnSent[learn.Slot] = true
		return learn.Val, learn.Slot, true
	}
	return val, sid, false
}

// TODO(student): Add any other unexported methods needed.
