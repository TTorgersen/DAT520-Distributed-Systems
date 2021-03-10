package failuredetector

import (
	"time"
)

// EvtFailureDetector represents a Eventually Perfect Failure Detector as
// described at page 53 in:
// Christian Cachin, Rachid Guerraoui, and Luís Rodrigues: "Introduction to
// Reliable and Secure Distributed Programming" Springer, 2nd edition, 2011.
type EvtFailureDetector struct {
	id        int          // the id of this node
	nodeIDs   []int        // node ids for every node in cluster
	alive     map[int]bool // map of node ids considered alive
	suspected map[int]bool // map of node ids  considered suspected

	sr SuspectRestorer // Provided SuspectRestorer implementation

	delay         time.Duration // the current delay for the timeout procedure
	delta         time.Duration // the delta value to be used when increasing delay
	timeoutSignal *time.Ticker  // the timeout procedure ticker

	hbSend chan<- Heartbeat // channel for sending outgoing heartbeat messages
	hbIn   chan Heartbeat   // channel for receiving incoming heartbeat messages
	stop   chan struct{}    // channel for signaling a stop request to the main run loop

	testingHook func() // DO NOT REMOVE THIS LINE. A no-op when not testing.
}

// NewEvtFailureDetector returns a new Eventual Failure Detector. It takes the
// following arguments:
//
// id: The id of the node running this instance of the failure detector.
//
// nodeIDs: A list of ids for every node in the cluster (including the node
// running this instance of the failure detector).
//
// ld: A leader detector implementing the SuspectRestorer interface.
//
// delta: The initial value for the timeout interval. Also the value to be used
// when increasing delay.
//
// hbSend: A send only channel used to send heartbeats to other nodes.
func NewEvtFailureDetector(id int, nodeIDs []int, sr SuspectRestorer, delta time.Duration, hbSend chan<- Heartbeat) *EvtFailureDetector {
	suspected := make(map[int]bool)
	alive := make(map[int]bool)

	// Setter alle nodes til alive
	for i := 0; i < len(nodeIDs); i++ {
		nr := nodeIDs[i]
		alive[nr] = true
	}
	//print(delta)

	// Starttimer (delay)

	return &EvtFailureDetector{
		id:        id,
		nodeIDs:   nodeIDs,
		alive:     alive,
		suspected: suspected,

		sr: sr,

		delay: delta,
		delta: delta,

		hbSend: hbSend,
		hbIn:   make(chan Heartbeat, 8),
		stop:   make(chan struct{}),

		testingHook: func() {}, // DO NOT REMOVE THIS LINE. A no-op when not testing.
	}
}

// Start starts e's main run loop as a separate goroutine. The main run loop
// handles incoming heartbeat requests and responses. The loop also trigger e's
// timeout procedure at an interval corresponding to e's internal delay
// duration variable.
func (e *EvtFailureDetector) Start() {
	e.timeoutSignal = time.NewTicker(e.delay)
	go func() {
		for {
			e.testingHook() // DO NOT REMOVE THIS LINE. A no-op when not testing.
			select {
			case hbReq := <-e.hbIn: // check if we get a heart beat request
				// Handle incoming heartbeat. If request is true, then make a reply
				// Sent the reply on the send chanel
				if hbReq.Request == true {
					hbReply := Heartbeat{To: hbReq.From, From: hbReq.To, Request: false}
					e.hbSend <- hbReply
				} else if hbReq.Request == false {
					e.alive[hbReq.From] = true
				}
			case <-e.timeoutSignal.C:
				e.timeout()
			case <-e.stop:
				return
			}
		}
	}()
}

// DeliverHeartbeat delivers heartbeat hb to failure detector e.
func (e *EvtFailureDetector) DeliverHeartbeat(hb Heartbeat) {
	e.hbIn <- hb
}

// Stop stops e's main run loop.
func (e *EvtFailureDetector) Stop() {
	e.stop <- struct{}{}
}

// Internal: timeout runs e's timeout procedure.
func (e *EvtFailureDetector) timeout() {
	// TODO(student): Implement timeout procedure
	// Check if alive intersect suspected and increse delay
	for i := range e.alive {
		if e.alive[i] == true && e.suspected[i] == true {
			e.delay += e.delta
			break // Only one intersection to increse delta
		}
	}
	for val := range e.nodeIDs {
		// If not alive and not suspected, then add to suspected
		if e.alive[val] == false && e.suspected[val] == false {
			e.suspected[val] = true
			e.sr.Suspect(val)
		}
		// If alive and suspected then delete suspected note from suspected
		if e.alive[val] == true && e.suspected[val] == true {
			//e.suspected[i] = false
			delete(e.suspected, val)
			e.sr.Restore(val)
		}
		hbReply := Heartbeat{To: val, From: e.id, Request: true}
		e.hbSend <- hbReply
	}
	e.alive = make(map[int]bool)
	e.Start()
}
