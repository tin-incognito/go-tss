package network

import (
	"fmt"
)

type Chain struct {
	TxsQueue chan struct{}
	stopCh   chan struct{}
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
		case Tx := <-c.TxsQueue:
			fmt.Println(Tx)
		}
	}
}

func (c *Chain) scan() error {
	//TODO: scan external network here then return value to TxsQueue channel
	return nil
}
