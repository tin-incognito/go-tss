package network

import (
	"fmt"
)

type ChainConfig struct{}

type Chain struct {
	ShieldTxsQueue   chan struct{} // TODO: @thach define txs (tx in external network)
	UnshieldTxsQueue chan struct{} // TODO: @thach define txs (tx from Incognitochain)
	cfg              ChainConfig
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
	//TODO: @thach scan external network here then return value to TxsQueue channel

	// call json rpc to external network full node (from external network)
	// catch event deposit
	//c.ShieldTxsQueue <- t

	// sync latest state beacon (from Incognitochain)
	//c.UnshieldTxsQueue <- t

	// receive external network tx
	//t := struct{}{}
	return nil
}
