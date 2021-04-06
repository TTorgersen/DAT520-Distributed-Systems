package multipaxos

import (
	"container/list"
	detector "dat520/lab3/detector"
	"fmt"
	"sort"
	"time"
)

// Proposer represents a proposer as defined by the Multi-Paxos algorithm.
type Proposer struct {
	id     int
	quorum int
	n      int

	crnd     Round
	adu      SlotID
	nextSlot SlotID

	promises     []*Promise
	promiseCount int

	phaseOneDone           bool
	phaseOneProgressTicker *time.Ticker

	acceptsOut *list.List
	requestsIn *list.List

	ld     detector.LeaderDetector
	leader int

	prepareOut chan<- Prepare
	acceptOut  chan<- Accept
	promiseIn  chan Promise
	cvalIn     chan Value

	incDcd chan struct{}
	stop   chan struct{}
}

type votedValues struct {
	Vrnd Round
	Vval Value
}

// NewProposer returns a new Multi-Paxos proposer. It takes the following
// arguments:
//
// id: The id of the node running this instance of a Paxos proposer.
//
// nrOfNodes: The total number of Paxos nodes.
//
// adu: all-decided-up-to. The initial id of the highest _consecutive_ slot
// that has been decided. Should normally be set to -1 initially, but for
// testing purposes it is passed in the constructor.
//
// ld: A leader detector implementing the detector.LeaderDetector interface.
//
// prepareOut: A send only channel used to send prepares to other nodes.
//
// The proposer's internal crnd field should initially be set to the value of
// its id.
func NewProposer(id, nrOfNodes, adu int, ld detector.LeaderDetector, prepareOut chan<- Prepare, acceptOut chan<- Accept) *Proposer {
	return &Proposer{
		id:     id,
		quorum: (nrOfNodes / 2) + 1,
		n:      nrOfNodes,

		crnd:     Round(id),
		adu:      SlotID(adu),
		nextSlot: 0,

		promises: make([]*Promise, nrOfNodes),

		phaseOneProgressTicker: time.NewTicker(time.Second * 5),

		acceptsOut: list.New(),
		requestsIn: list.New(),

		ld:     ld,
		leader: ld.Leader(),

		prepareOut: prepareOut,
		acceptOut:  acceptOut,
		promiseIn:  make(chan Promise, 2000000),
		cvalIn:     make(chan Value, 2000000),

		incDcd: make(chan struct{}),
		stop:   make(chan struct{}),
	}
}

// Start starts p's main run loop as a separate goroutine.
func (p *Proposer) Start() {
	trustMsgs := p.ld.Subscribe()
	go func() {
		for {
			select {
			case prm := <-p.promiseIn:
				fmt.Println("Promise received")
				accepts, output := p.handlePromise(prm)
				if !output {
					continue
				}
				p.nextSlot = p.adu + 1
				p.acceptsOut.Init()
				p.phaseOneDone = true
				fmt.Println("phase1 is true")
				for _, acc := range accepts {
					p.acceptsOut.PushBack(acc)
				}
				p.sendAccept()
			case cval := <-p.cvalIn:
				fmt.Println("Client value received in proposer")
				if p.id != p.leader {
					continue
				}
				p.requestsIn.PushBack(cval)
				if !p.phaseOneDone {
					continue
				}
				p.sendAccept()
			case <-p.incDcd:
				p.adu++
				if p.id != p.leader {
					continue
				}
				if !p.phaseOneDone {
					continue
				}
				p.sendAccept()
			case <-p.phaseOneProgressTicker.C:
				fmt.Println("Got into the phase one chanel")
				if p.id == p.leader && !p.phaseOneDone {
					fmt.Println("Starting phase 1")
					p.startPhaseOne()
				}
			case leader := <-trustMsgs:
				fmt.Println("Leader got trust message")
				p.leader = leader
				if leader == p.id {
					fmt.Println("Starting phase 1")
					p.startPhaseOne()
				}
			case <-p.stop:
				return
			}
		}
	}()
}

// Stop stops p's main run loop.
func (p *Proposer) Stop() {
	p.stop <- struct{}{}
}

// DeliverPromise delivers promise prm to proposer p.
func (p *Proposer) DeliverPromise(prm Promise) {
	p.promiseIn <- prm
}

// DeliverClientValue delivers client value cval from to proposer p.
func (p *Proposer) DeliverClientValue(cval Value) {
	p.cvalIn <- cval
}

// IncrementAllDecidedUpTo increments the Proposer's adu variable by one.
func (p *Proposer) IncrementAllDecidedUpTo() {
	p.incDcd <- struct{}{}
}

// Internal: handlePromise processes promise prm according to the Multi-Paxos
// algorithm. If handling the promise results in proposer p emitting a
// corresponding accept slice, then output will be true and accs contain the
// accept messages. If handlePromise returns false as output, then accs will be
// a nil slice.
func (p *Proposer) handlePromise(prm Promise) (accs []Accept, output bool) {
	// TODO(student)
	//Spec 1 - Ignore if promise round is different from proposers round
	if prm.Rnd != p.crnd {
		return nil, false
	}
	//Spec 2 - Ignore if it has previously received a promise from the same node for the same round
	for _, promise := range p.promises {
		if promise == nil {
			continue
		}
		if promise.From == prm.From && promise.Rnd == prm.Rnd {
			return nil, false
		}

	}
	// Append promise to slice of promises
	p.promises = append(p.promises, &prm)
	promisesNotNil := []*Promise{}
	// Count promises not equal to zero
	for _, promise := range p.promises {
		if promise != nil {
			promisesNotNil = append(promisesNotNil, promise)
		}
	}
	// If number of promises not equal to nil is a qourum
	if len(promisesNotNil) < p.quorum {
		return nil, false
	}
	// Spec 5 - If a PromiseSlot in promise message is for a slot lower than the Proposer's current adu (all-decided-up-to),
	// then the PromiseSlot should be ignored
	accs = []Accept{}
	promiseSlot := map[SlotID]votedValues{}
	// Iterate over all promises not nil
	for _, prom := range promisesNotNil {
		// Iterate over all PromiseSlots
		for _, slot := range prom.Slots {
			// Ignore PromiseSlot with ID smaller or equal to adu
			if slot.ID > p.adu {
				// Check if the slot ID is present in the promiseSlot
				if v, OK := promiseSlot[slot.ID]; OK {
					// if present, check if the existing PromiseSlot Vrnd is smaller. If smaller, then update
					if v.Vrnd < slot.Vrnd {
						promiseSlot[slot.ID] = votedValues{Vrnd: slot.Vrnd, Vval: slot.Vval}
					}
				} else {
					promiseSlot[slot.ID] = votedValues{Vrnd: slot.Vrnd, Vval: slot.Vval}
				}
			}
		}
	}

	// Append the result from spec 5 to accs slice
	for key, value := range promiseSlot {
		accs = append(accs, Accept{From: p.id, Slot: key, Rnd: p.crnd, Val: value.Vval})
	}
	// Output: If handling input result in a quorum for the current round, then accs should contain a slice of accept
	// messages for the slots the Proposer is bound in. If the proposer is not bounded in any slot the accs should be an
	// empty slice
	if len(accs) == 0 {
		return []Accept{}, true
	}
	// Spec 3 - All Accept messages in the accs slice should be in increased order
	sort.SliceStable(accs, func(i, j int) bool {
		return accs[i].Slot < accs[j].Slot
	})
	// Spec 4 - If there is a gap in the set of slots, then set no-op value in gap
	for i := 0; i < len(accs)-1; i++ {
		for accs[i].Slot+1 != accs[i+1].Slot {
			newID := accs[i].Slot + 1
			noopaccept := Accept{From: p.id, Slot: newID, Rnd: p.crnd, Val: Value{Noop: true}}
			accs = append(accs, Accept{})
			copy(accs[i+2:], accs[i+1:])
			accs[i+1] = noopaccept
		}
	}
	return accs, true
}

// Internal: increaseCrnd increases proposer p's crnd field by the total number
// of Paxos nodes.
func (p *Proposer) increaseCrnd() {
	p.crnd = p.crnd + Round(p.n)
}

// Internal: startPhaseOne resets all Phase One data, increases the Proposer's
// crnd and sends a new Prepare with Slot as the current adu.
func (p *Proposer) startPhaseOne() {
	p.phaseOneDone = false
	p.promises = make([]*Promise, p.n)
	p.increaseCrnd()
	p.prepareOut <- Prepare{From: p.id, Slot: p.adu, Crnd: p.crnd}
}

// Internal: sendAccept sends an accept from either the accept out queue
// (generated after Phase One) if not empty, or, it generates an accept using a
// value from the client request queue if not empty. It does not send an accept
// if the previous slot has not been decided yet.
func (p *Proposer) sendAccept() {
	const alpha = 1
	if !(p.nextSlot <= p.adu+alpha) {
		fmt.Println("pri 0")
		// We must wait for the next slot to be decided before we can
		// send an accept.
		//
		// For Lab 6: Alpha has a value of one here. If you later
		// implement pipelining then alpha should be extracted to a
		// proposer variable (alpha) and this function should have an
		// outer for loop.
		return
	}

	// Pri 1: If bounded by any accepts from Phase One -> send previously
	// generated accept and return.
	if p.acceptsOut.Len() > 0 {
		fmt.Println("pri 1")
		acc := p.acceptsOut.Front().Value.(Accept)
		p.acceptsOut.Remove(p.acceptsOut.Front())
		p.acceptOut <- acc
		p.nextSlot++
		return
	}

	// Pri 2: If any client request in queue -> generate and send
	// accept.
	if p.requestsIn.Len() > 0 {
		fmt.Println("pri 2")
		cval := p.requestsIn.Front().Value.(Value)
		p.requestsIn.Remove(p.requestsIn.Front())
		acc := Accept{
			From: p.id,
			Slot: p.nextSlot,
			Rnd:  p.crnd,
			Val:  cval,
		}
		p.nextSlot++
		fmt.Printf("sending acc to acceptout")
		p.acceptOut <- acc
	}
}
