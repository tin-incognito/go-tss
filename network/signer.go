package network

import "sync"

type Signer struct {
	wg *sync.WaitGroup
}

func NewSigner() (*Signer, error) {
	res := &Signer{
		wg: &sync.WaitGroup{},
	}
	return res, nil
}

func (s *Signer) Start() error {
	s.wg.Add(1)
	go s.processTxnOut()

	s.wg.Add(1)
	go s.processKeygen()

	s.wg.Add(1)
	go s.signTransactions()

	//s.blockScanner.Start(nil)

	return nil
}

func (s *Signer) processTxnOut() {

}

func (s *Signer) processKeygen() {

}

func (s *Signer) signTransactions() {

}
