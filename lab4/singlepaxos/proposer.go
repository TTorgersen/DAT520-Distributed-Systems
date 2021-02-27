package singlepaxos

// Proposer represents a proposer as defined by the single-decree Paxos
// algorithm.
type Proposer struct {
	crnd        Round
	clientValue Value
	ID          int
	nrOfNodes   int
	quorum      int
	promises    map[int]*Promise

	// TODO(student): algorithm implementation
	// Add other needed fields
}

// NewProposer returns a new single-decree Paxos proposer.
// It takes the following arguments:
//
// id: The id of the node running this instance of a Paxos proposer.
//
// nrOfNodes: The total number of Paxos nodes.
//
// The proposer's internal crnd field should initially be set to the value of
// its id.
func NewProposer(id int, nrOfNodes int) *Proposer {
	// TODO(student): algorithm and distributed implementation
	return &Proposer{
		crnd:      Round(id),
		nrOfNodes: nrOfNodes,
		ID:        id,
		quorum:    (nrOfNodes / 2) + 1,
		promises:  make(map[int]*Promise),
	}
}

// Internal: handlePromise processes promise prm according to the single-decree
// Paxos algorithm. If handling the promise results in proposer p emitting a
// corresponding accept, then output will be true and acc contain the promise.
// If handlePromise returns false as output, then acc will be a zero-valued
// struct.
func (p *Proposer) handlePromise(prm Promise) (acc Accept, output bool) {
	// TODO(student): algorithm implementation
	if prm.Rnd == p.crnd { //on <PROMISE, rnd, vrnd, vval> with rnd = crnd from acceptor a
		p.promises[prm.From] = &prm      //add the promise to promises
		if len(p.promises) >= p.quorum { //if the length of promises is larger than half the nodes
			promiseValue := false
			for _, promise := range p.promises { //go through list and see if there are any promises with value
				if promise.Vval != ZeroValue {
					promiseValue = true
				}
			}
			if promiseValue { //if there is a promise with value, pick the largest
				p.clientValue = p.pickLargest()
			}
			return Accept{From: p.ID, Rnd: p.crnd, Val: p.clientValue}, true
		}
	}
	return Accept{}, false
}

// Internal: increaseCrnd increases proposer p's crnd field by the total number
// of Paxos nodes.
func (p *Proposer) increaseCrnd() {
	// TODO(student): algorithm implementation
	p.crnd = p.crnd + Round(p.nrOfNodes)
}

// Function to pick the value from the largest vrnds
func (p *Proposer) pickLargest() Value {
	vrnds := -1
	in := -1
	for i, prom := range p.promises {
		if prom.Vrnd > Round(vrnds) {
			vrnds = int(prom.Vrnd)
			in = i
		}
	}
	return p.promises[in].Vval
}

// TODO(student): Add any other unexported methods needed.
