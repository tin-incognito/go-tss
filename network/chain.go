package network

import (
	"fmt"
)

type Chain struct {
	ShieldTxsQueue   chan struct{} // TODO: define txs
	UnshieldTxsQueue chan struct{} // TODO: define txs
	stopCh           chan struct{}
}

func InitChains() map[int]*Chain {
	res := make(map[int]*Chain) // 0 -> Incognitochain, 1 -> BTC, 2 -> ETH
	return res
}

func (c *Chain) Start() error {
	go c.scan()
	go c.processTxIns()
	return nil
}

func (c *Chain) processTxIns() {
	for {
		select {
		case <-c.stopCh:
			return
		case Tx := <-c.ShieldTxsQueue: // external network
			fmt.Println(Tx)
		case Tx := <-c.UnshieldTxsQueue: // Incognitochain
			fmt.Println(Tx)
		}
	}
}

func (c *Chain) scan() error {
	//TODO: scan external network here then return value to TxsQueue channel

	// receive external network tx
	//t := struct{}{}
	//c.ShieldTxsQueue <- t
	//c.UnshieldTxsQueue <- t
	return nil
}
