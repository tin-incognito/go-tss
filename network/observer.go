package network

import (
	"fmt"
	"sync"

	"gitlab.com/thorchain/tss/go-tss/network/chain"
)

type Observer struct {
	wg     *sync.WaitGroup
	chains map[string]chain.Chain // eth -> chain object, btc -> chain object
}

func NewObserver(chains map[string]chain.Chain) (*Observer, error) {
	res := &Observer{
		wg:     &sync.WaitGroup{},
		chains: chains,
	}
	return res, nil
}

func (o *Observer) Start() error {
	fmt.Println("Start observer")
	for _, v := range o.chains {
		v.Start()
	}
	return nil
}
