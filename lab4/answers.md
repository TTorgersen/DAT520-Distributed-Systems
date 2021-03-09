## Answers to Paxos Questions 

You should write down your answers to the
[questions](README.md#questions)
for Lab 4 in this file. 

1. A naive implementation is not guaranteed to avoid this problem. This could happen if the nodes never reaches consensus, and that the proposers just keeping requesting new values. But a solution could be to route the requests to a designated leader that acts as proposer.

2. The value in the Prepare message contains the id of the Node that wants to propose a new value.

3. Your answer.

4. Your answer.

5. Your answer.

6. Your answer.

7. Your answer.

8. Yes an acceptor can accept value one in round 1, and will accept the second value in round 2 since the round number is higher, and therefore has a higher priority. It will then use value two as the current value that will be sent to learners.

9. A value is chosen when a single proposal with that value has been accepted by a majoriy of the acceptors. To learn that a value has been chosen, a learner must find out that a proposal has been accepted by majority of acceptors. After this has been confirmed, the chosen value is learned and can be sent to clients.
