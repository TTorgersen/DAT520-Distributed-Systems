package multipaxos

import (
	"sort"
)

// Acceptor represents an acceptor as defined by the Multi-Paxos algorithm.
type Acceptor struct { // TODO(student): algorithm and distributed implementation
	// Add needed fields
	ID      int
	rnd     Round //Current round number
	Slots   []PromiseSlot
	MaxSlot SlotID

	PromiseOutChan chan<- Promise
	LearnOutChan   chan<- Learn
	stopChan       chan struct{}
	PrepareChan    chan Prepare
	AcceptChan     chan Accept
}

// NewAcceptor returns a new Multi-Paxos acceptor.
// It takes the following arguments:
//
// id: The id of the node running this instance of a Paxos acceptor.
//
// promiseOut: A send only channel used to send promises to other nodes.
//
// learnOut: A send only channel used to send learns to other nodes.
func NewAcceptor(id int, promiseOut chan<- Promise, learnOut chan<- Learn) *Acceptor {
	// TODO(student): algorithm and distributed implementation
	return &Acceptor{
		ID:             id,
		rnd:            NoRound,
		MaxSlot:        -1,
		Slots:          []PromiseSlot{},
		PromiseOutChan: promiseOut,
		LearnOutChan:   learnOut,
		stopChan:       make(chan struct{}),
		PrepareChan:    make(chan Prepare, 2000000),
		AcceptChan:     make(chan Accept, 2000000),
	}
}

// Start starts a's main run loop as a separate goroutine. The main run loop
// handles incoming prepare and accept messages.
func (a *Acceptor) Start() {
	go func() {
		for {
			// TODO(student): distributed implementation
			select {
			case prpMsg := <-a.PrepareChan:
				if prmMsg, sendMsg := a.handlePrepare(prpMsg); sendMsg == true {
					a.PromiseOutChan <- prmMsg
				}
			case accMsg := <-a.AcceptChan: //TODO: Create a channel to receive accept messages from other nodes
				if lrnMsg, sendMsg := a.handleAccept(accMsg); sendMsg == true {
					a.LearnOutChan <- lrnMsg
				}
			case <-a.stopChan:
				break
			}
		}
	}()
}

// Stop stops a's main run loop.
func (a *Acceptor) Stop() {
	// TODO(student): distributed implementation
	a.stopChan <- struct{}{}
}

// DeliverPrepare delivers prepare prp to acceptor a.
func (a *Acceptor) DeliverPrepare(prp Prepare) {
	// TODO(student): distributed implementation
	a.PrepareChan <- prp
}

// DeliverAccept delivers accept acc to acceptor a.
func (a *Acceptor) DeliverAccept(acc Accept) {
	// TODO(student): distributed implementation
	a.AcceptChan <- acc
}



// Internal: handlePrepare processes prepare prp according to the Multi-Paxos
// algorithm. If handling the prepare results in acceptor a emitting a
// corresponding promise, then output will be true and prm contain the promise.
// If handlePrepare returns false as output, then prm will be a zero-valued
// struct.
func (a *Acceptor) handlePrepare(prp Prepare) (prm Promise, output bool) {
	// TODO(student): algorithm implementation
	if prp.Crnd > a.rnd { //<PREPARE, n> with n > rnd from proposer c
		a.rnd = prp.Crnd
		var promiseSlots []PromiseSlot // Make a slice for promiseslot
		// Loop through and add every slot with ID larger than prepare SlotID
		for _, slot := range a.Slots {
			if slot.ID >= prp.Slot {
				promiseSlots = append(promiseSlots, slot)
			}
		}
		return Promise{To: prp.From, From: a.ID, Rnd: a.rnd, Slots: promiseSlots}, true
	}
	return Promise{}, false
}

// Internal: handleAccept processes accept acc according to the Multi-Paxos
// algorithm. If handling the accept results in acceptor a emitting a
// corresponding learn, then output will be true and lrn contain the learn.  If
// handleAccept returns false as output, then lrn will be a zero-valued struct.
func (a *Acceptor) handleAccept(acc Accept) (lrn Learn, output bool) {
	// TODO(student): algorithm implementation
	if acc.Rnd >= a.rnd {
		a.rnd = acc.Rnd //Update the round
		// Loop through the slice and check if ID in slice is equal to accept SlotID
		for i, slot := range a.Slots {
			// If ID is equal to SlotID, uppdate the PromiseSlot with round and value
			if slot.ID == acc.Slot {
				slot.Vrnd = acc.Rnd
				slot.Vval = acc.Val
				a.Slots[i].Vrnd = acc.Rnd
				a.Slots[i].Vval = acc.Val
				return Learn{From: a.ID, Slot: slot.ID, Rnd: slot.Vrnd, Val: slot.Vval}, true
			}
		}
		// Append to the slice and sort the slice
		a.Slots = append(a.Slots, PromiseSlot{ID: acc.Slot, Vrnd: acc.Rnd, Vval: acc.Val})
		sort.SliceStable(a.Slots, func(i, j int) bool {
			return a.Slots[i].ID < a.Slots[j].ID
		})
		//type Learn struct{From int; Slot SlotID; Rnd Round; Val Value}
		return Learn{From: a.ID, Slot: acc.Slot, Rnd: acc.Rnd, Val: acc.Val}, true
	}
	return Learn{}, false
}

// TODO(student): Add any other unexported methods needed.
