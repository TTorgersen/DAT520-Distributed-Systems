package singlepaxos

// Learner represents a learner as defined by the single-decree Paxos
// algorithm.
type Learner struct { // TODO(student): algorithm implementation
	// Add needed fields
	// Tip: you need to keep the decided values by the Paxos nodes somewhere
	id         int
	nrOfNodes  int
	messages   map[int]*Learn
	rndMax     int
	curVal     string
	valCounter int
	prevNode   int
}

// NewLearner returns a new single-decree Paxos learner. It takes the
// following arguments:
//
// id: The id of the node running this instance of a Paxos learner.
//
// nrOfNodes: The total number of Paxos nodes.
func NewLearner(id int, nrOfNodes int) *Learner {
	// TODO(student): algorithm implementation
	newLearner := &Learner{id: id, nrOfNodes: nrOfNodes, rndMax: 0, curVal: "none", valCounter: 0, prevNode: -1} //declare new learner
	newLearner.messages = make(map[int]*Learn)

	return newLearner
}

// Internal: handleLearn processes learn lrn according to the single-decree
// Paxos algorithm. If handling the learn results in learner l emitting a
// corresponding decided value, then output will be true and val contain the
// decided value. If handleLearn returns false as output, then val will have
// its zero value.
func (l *Learner) handleLearn(learn Learn) (val Value, output bool) {
	// TODO(student): algorithm implementation
	majority := (l.nrOfNodes / 2) + 1
	l.messages[learn.From] = &learn

	// 1. Sjekke hvilke verdier første node har
	// 2. Er neste node på samme runde, eller høyere runde?
	// Hvis høyere runde, overskriv de forrige verdiene
	//Hvis samme runde, er det samme node eller ny node? Return false
	// 3. Sjekk hvor ofte de samme verdiene på samme runde dukker opp
	// 4. Sjekk om majoriteten av nodene har samme verdi på samme runde

	if l.prevNode != learn.From { //ER det samme node som forrige?
		l.prevNode = learn.From
	} else {
		return ZeroValue, false
	}

	if int(learn.Rnd) > l.rndMax { //Which is the highest round
		l.rndMax = int(learn.Rnd)
		l.curVal = string(learn.Val)
		l.valCounter = 0
	}

	switch len(l.messages) {
	case 0:
		return ZeroValue, false
	case 1:
		if l.rndMax == int(learn.Rnd) && learn.Val == Value(l.curVal) {
			l.valCounter++
			if l.valCounter >= majority {
				return Value(l.curVal), true
			}
		}
	case 2:
		if l.rndMax == int(learn.Rnd) && learn.Val == Value(l.curVal) {
			l.valCounter++
			if l.valCounter >= majority {
				return Value(l.curVal), true
			}
		}
	case 3:
		if l.rndMax == int(learn.Rnd) && learn.Val == Value(l.curVal) {
			l.valCounter++
			if l.valCounter >= majority {
				return Value(l.curVal), true
			}
		}
	default:
		return ZeroValue, false
	}
	return ZeroValue, false
}

// TODO(student): Add any other unexported methods needed.
