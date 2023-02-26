package network

import "sync"

type Observer struct {
	wg *sync.WaitGroup
}

func NewObserver() (*Observer, error) {
	res := &Observer{
		wg: &sync.WaitGroup{},
	}
	return res, nil
}

func (o *Observer) Start() error {
	go o.processTxIns()
	return nil
}

func (o *Observer) processTxIns() {

}
