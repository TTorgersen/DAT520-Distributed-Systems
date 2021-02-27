package singlepaxos

// Acceptor represents an acceptor as defined by the single-decree Paxos
// algorithm.
type Acceptor struct { // TODO(student): algorithm implementation
	// Add needed fields
	ID   int
	rnd  Round //Current round number
	vrnd Round // Last voted round
	vval Value // Value of last voted round
}

// NewAcceptor returns a new single-decree Paxos acceptor.
// It takes the following arguments:
//
// id: The id of the node running this instance of a Paxos acceptor.
func NewAcceptor(id int) *Acceptor {
	// TODO(student): algorithm implementation
	return &Acceptor{
		ID:   id,
		rnd:  NoRound,
		vrnd: NoRound,
		vval: ZeroValue,
	}
}

// Internal: handlePrepare processes prepare prp according to the single-decree
// Paxos algorithm. If handling the prepare results in acceptor a emitting a
// corresponding promise, then output will be true and prm contain the promise.
// If handlePrepare returns false as output, then prm will be a zero-valued
// struct.
func (a *Acceptor) handlePrepare(prp Prepare) (prm Promise, output bool) {
	// TODO(student): algorithm implementation
	if prp.Crnd > a.rnd { //<PREPARE, n> with n > rnd from proposer c
		a.rnd = prp.Crnd
		return Promise{To: prp.From, From: a.ID, Rnd: a.rnd, Vrnd: a.vrnd, Vval: a.vval}, true
	}
	return Promise{}, false
}

// Internal: handleAccept processes accept acc according to the single-decree
// Paxos algorithm. If handling the accept results in acceptor a emitting a
// corresponding learn, then output will be true and lrn contain the learn.  If
// handleAccept returns false as output, then lrn will be a zero-valued struct.
func (a *Acceptor) handleAccept(acc Accept) (lrn Learn, output bool) {
	// TODO(student): algorithm implementation
	if acc.Rnd >= a.rnd && acc.Rnd != a.vrnd { //on <ACCEPT, n, v> with n >= rnd ^ n != vrnd from proposer c
		a.rnd = acc.Rnd
		a.vrnd = acc.Rnd
		a.vval = acc.Val
		return Learn{From: a.ID, Rnd: a.rnd, Val: a.vval}, true
	}
	return Learn{}, false
}

// TODO(student): Add any other unexported methods needed.
