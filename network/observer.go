package network

import (
	"fmt"
	"sync"
)

type Observer struct {
	wg     *sync.WaitGroup
	chains map[int]*Chain // eth -> chain object, btc -> chain object
}

func NewObserver(chains map[int]*Chain) (*Observer, error) {
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
