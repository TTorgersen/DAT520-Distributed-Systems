package multipaxos

// Learner represents a learner as defined by the Multi-Paxos algorithm.
type Learner struct { // TODO(student): algorithm and distributed implementation
	// Add needed fields
	id         int
	nrOfNodes  int
	decidedOut chan string
	messages   map[int]*Learn
	rndMax     int
	curVal     Value
	valCounter int
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
	newLearner := &Learner{id: id, nrOfNodes: nrOfNodes, rndMax: 0, curVal: Value{ClientID: "0000", ClientSeq: -10, Command: "none"}, valCounter: -1}
	newLearner.messages = make(map[int]*Learn)
	newLearner.decidedOut = make(chan string)
	return newLearner
}

// Start starts l's main run loop as a separate goroutine. The main run loop
// handles incoming learn messages.
func (l *Learner) Start() {
	go func() {
		for {
			// TODO(student): distributed implementation
		}
	}()
}

// Stop stops l's main run loop.
func (l *Learner) Stop() {
	// TODO(student): distributed implementation
}

// DeliverLearn delivers learn lrn to learner l.
func (l *Learner) DeliverLearn(lrn Learn) {
	// TODO(student): distributed implementation
}

// Internal: handleLearn processes learn lrn according to the Multi-Paxos
// algorithm. If handling the learn results in learner l emitting a
// corresponding decided value, then output will be true, sid the id for the
// slot that was decided and val contain the decided value. If handleLearn
// returns false as output, then val and sid will have their zero value.
func (l *Learner) handleLearn(learn Learn) (val Value, sid SlotID, output bool) {
	// TODO(student): algorithm implementation
	majority := (l.nrOfNodes / 2) + 1
	l.messages[learn.From] = &learn

	if int(learn.Rnd) > l.rndMax { //Which is the highest round
		l.rndMax = int(learn.Rnd)
		l.curVal = Value(learn.Val)
		l.valCounter = 0
	}

	switch len(l.messages) {
	case 0:
		return Value{}, 0, false
	case 1:
		if l.rndMax == int(learn.Rnd) && learn.Val == Value(l.curVal) {
			l.valCounter++
			if l.valCounter >= majority {
				return Value(l.curVal), 1, true
			}
		}
	case 2:
		if l.rndMax == int(learn.Rnd) && learn.Val == Value(l.curVal) {
			l.valCounter++
			if l.valCounter >= majority {
				return Value(l.curVal), 1, true
			}
		}
	case 3:
		if l.rndMax == int(learn.Rnd) && learn.Val == Value(l.curVal) {
			l.valCounter++
			if l.valCounter >= majority {
				return Value(l.curVal), 1, true
			}
		}
	case 4:
		if l.rndMax == int(learn.Rnd) && learn.Val == Value(l.curVal) {
			l.valCounter++
			if l.valCounter >= majority {
				return Value(l.curVal), 1, true
			}
		}
	case 5:
		if l.rndMax == int(learn.Rnd) && learn.Val == Value(l.curVal) {
			l.valCounter++
			if l.valCounter >= majority {
				return Value(l.curVal), 1, true
			}
		}
	default:
		return Value{ClientID: "-1", ClientSeq: -1, Command: "-1"}, 0, false

	}

	return Value{ClientID: "-1", ClientSeq: -1, Command: "-1"}, 0, false
}

// TODO(student): Add any other unexported methods needed.
