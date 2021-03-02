package multipaxos

import (
	"container/list"
	"dat520/lab3/leaderdetector"
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
	promisesNotNill []*Promise
	promiseCount int

	phaseOneDone           bool
	phaseOneProgressTicker *time.Ticker

	acceptsOut *list.List
	requestsIn *list.List

	ld     leaderdetector.LeaderDetector
	leader int

	prepareOut chan<- Prepare
	acceptOut  chan<- Accept
	promiseIn  chan Promise
	cvalIn     chan Value

	incDcd chan struct{}
	stop   chan struct{}
}

type votedValues struct{
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
func NewProposer(id, nrOfNodes, adu int, ld leaderdetector.LeaderDetector, prepareOut chan<- Prepare, acceptOut chan<- Accept) *Proposer {
	return &Proposer{
		id:     id,
		quorum: (nrOfNodes / 2) + 1,
		n:      nrOfNodes,

		crnd:     Round(id),
		adu:      SlotID(adu),
		nextSlot: 0,

		promises: make([]*Promise, nrOfNodes),

		phaseOneProgressTicker: time.NewTicker(time.Second),

		acceptsOut: list.New(),
		requestsIn: list.New(),

		ld:     ld,
		leader: ld.Leader(),

		prepareOut: prepareOut,
		acceptOut:  acceptOut,
		promiseIn:  make(chan Promise, 8),
		cvalIn:     make(chan Value, 8),

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
				accepts, output := p.handlePromise(prm)
				if !output {
					continue
				}
				p.nextSlot = p.adu + 1
				p.acceptsOut.Init()
				p.phaseOneDone = true
				for _, acc := range accepts {
					p.acceptsOut.PushBack(acc)
				}
				p.sendAccept()
			case cval := <-p.cvalIn:
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
				if p.id == p.leader && !p.phaseOneDone {
					p.startPhaseOne()
				}
			case leader := <-trustMsgs:
				p.leader = leader
				if leader == p.id {
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
	
	// if the round is different from our round, we simply ignore it.
	if prm.Rnd != p.crnd{return nil, false} //The Proposer should ignore a promise message if the promise has a round different from the Proposer's current round, 
	fmt.Println("CAN WE GO HERE?1")
	// The Proposer should ignore a promise message if it has previously received a promise from the same node for the same round.
	for _, prom := range p.promises{
		if prom == nil{continue} // if prom is nil we are done and should continue (Unsure if we need this)
		if prom.Rnd == p.crnd && prom.From == prm.From{return nil, false}
	}
	
	// if none of these has fired, we assume we need the promise and must append it to promise list.
	
	
	p.promises = append(p.promises, &prm)
	p.promisesNotNill = nil
	for _, prom := range p.promises{
		if prom != nil{
			p.promisesNotNill = append(p.promisesNotNill, prom)
		}
	}
	p.promises = p.promisesNotNill
	fmt.Println(p.promises)
	if len(p.promises)< p.quorum{
		return nil, false
	}
	//fmt.Println("CAN WE GO HERE?3")
	// if we are here we have a majority
	// now we need to get all acceptors
	accs = []Accept{}
	//fmt.Println("CAN WE GO HERE?4")
	promiseSlot := map[SlotID]votedValues{}
	
	for _, prom := range p.promises{
		for _, slot := range prom.Slots{
			if slot.ID >= p.adu{
				//fmt.Println("CAN WE GO HERE?4.6")
				// If a PromiseSlot in a promise message is for a slot lower than the Proposer's current adu (all-decided-up-to), then the PromiseSlot should be ignored.
				// we add slot to list


				var newTupple votedValues
				//fmt.Println("CAN WE GO HERE?4.7")
				newTupple.Vrnd = slot.Vrnd
				newTupple.Vval = slot.Vval
				if v, OK := promiseSlot[slot.ID]; OK{
					if v.Vrnd < slot.Vrnd{ //new round is higher than old round
						promiseSlot[slot.ID] = newTupple
					}
			
				} else{promiseSlot[slot.ID] = newTupple }


				promiseSlot[slot.ID] = newTupple
				//fmt.Println("CAN WE GO HERE?4.8")
			}

		}
	}
	//fmt.Println("CAN WE GO HERE?5")
	for key, value := range promiseSlot{
		
		accs = append(accs, Accept{From: p.id, Slot: key, Rnd: p.crnd, Val: value.Vval})
	}

	if len(accs) == 0{return []Accept{}, true}
	//fmt.Println("CAN WE GO HERE?6")
	// WE NEED TO SORT AND INSERT NOOP SLOTS
	sort.SliceStable(accs, func(i,j int) bool{return accs[i].Slot < accs[j].Slot})
	for i := 0; i < len(accs)-1; i++{
		for accs[i].Slot +1 != accs[i+1].Slot{
			newID := accs[i].Slot +1
			noopaccept := Accept{From: p.id, Slot: newID, Rnd: p.crnd, Val: Value{Noop:true,},}
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
		acc := p.acceptsOut.Front().Value.(Accept)
		p.acceptsOut.Remove(p.acceptsOut.Front())
		p.acceptOut <- acc
		p.nextSlot++
		return
	}

	// Pri 2: If any client request in queue -> generate and send
	// accept.
	if p.requestsIn.Len() > 0 {
		cval := p.requestsIn.Front().Value.(Value)
		p.requestsIn.Remove(p.requestsIn.Front())
		acc := Accept{
			From: p.id,
			Slot: p.nextSlot,
			Rnd:  p.crnd,
			Val:  cval,
		}
		p.nextSlot++
		p.acceptOut <- acc
	}
}
