package bankhandler

import (
	"dat520/lab5/bank"
	mp "dat520/lab5/multipaxos"
	"fmt"
)

//bankLearner struct
type BankHandler struct {
	adu                mp.SlotID
	bufferDecidedValue []mp.DecidedValue
	bankAccounts       map[int]bank.Account
	responseChanOut    chan<- mp.Response
	proposer           *mp.Proposer
}

// Function to create new banklearner
func NewBankHandler(responseChan chan<- mp.Response, proposer *mp.Proposer) *BankHandler {
	return &BankHandler{
		adu:                -1,
		bufferDecidedValue: []mp.DecidedValue{},
		bankAccounts:       map[int]bank.Account{},
		responseChanOut:    responseChan,
		proposer:           proposer,
	}
}


func (bh *BankHandler) SetBankAccounts(accounts map[int]bank.Account) {
	bh.bankAccounts = accounts
}

func (bh *BankHandler) GetBankAccounts() map[int]bank.Account{
	return bh.bankAccounts
}


func (bh *BankHandler) SetSlot(newSlot mp.SlotID) {
	bh.adu = newSlot
}



// Function to handle decided value
func (bh *BankHandler) HandleDecidedValue(dVal mp.DecidedValue) {
	// If slot id for value is larger than adu+1 then buffer value
	fmt.Println("In bankhandler, SlotID: ", dVal.SlotID, " bh.adu+1 : ", bh.adu+1)
	if dVal.SlotID > bh.adu+1 {
		fmt.Println("In a empty return ")
		bh.bufferDecidedValue = append(bh.bufferDecidedValue, dVal)
		return
	}
	// If the value is not a no-op
	fmt.Println("Dval.Value.noop", dVal.Value.Noop)
	if dVal.Value.Noop == false {
		// If account for account number in value is not found
		accountNum := dVal.Value.AccountNum
		if _, ok := bh.bankAccounts[accountNum]; ok != true {
			//Create and store new account with balance zero
			bh.bankAccounts[accountNum] = bank.Account{
				Number:  accountNum,
				Balance: 0,
			}
		}
		// Apply transaction from value to account if possible (e.g. user has blance)
		bankAccount := bh.bankAccounts[accountNum]
		transactionRes := bankAccount.Process(dVal.Value.Txn)
		// Update the bank account balance
		bh.bankAccounts[accountNum] = bankAccount
		// Create response with appropriate transaction results, client id and client seq
		response := mp.Response{
			ClientID:  dVal.Value.ClientID,
			ClientSeq: dVal.Value.ClientSeq,
			TxnRes:    transactionRes,
		}
		// Forward response to client handling module
		bh.responseChanOut <- response
	}
	bh.adu++
	bh.proposer.IncrementAllDecidedUpTo()
	// If has previously buffered value for (adu+1)
	for i, buffdecidedVal := range bh.bufferDecidedValue {
		if buffdecidedVal.SlotID == bh.adu+1 {
			bh.bufferDecidedValue = append(bh.bufferDecidedValue[:i], bh.bufferDecidedValue[i+1:]...)
			bh.HandleDecidedValue(buffdecidedVal)
		}
	}
}