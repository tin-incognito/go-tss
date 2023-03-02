package network

import (
	"bridge/x/bridge/types"
	"fmt"
	"sync"
	"time"

	"gitlab.com/thorchain/tss/go-tss/tss"
)

type Signer struct {
	wg            *sync.WaitGroup
	stopCh        chan struct{}
	BridgeScanner *BridgeScanner
	blockScanners map[int]*BlockScanner
	tssKeygen     *Keygen

	//logger        zerolog.Logger
}

func NewSigner(tssSever *tss.TssServer, blockUrl, stateUrl string) (*Signer, error) {
	res := &Signer{
		wg:            &sync.WaitGroup{},
		stopCh:        make(chan struct{}),
		BridgeScanner: NewBridgeScanner(blockUrl, stateUrl),
		blockScanners: make(map[int]*BlockScanner),
		tssKeygen:     NewTssKeygen(tssSever),
	}
	return res, nil
}

func (s *Signer) Start() error {
	fmt.Println("Start signer")
	go s.processTxnOut()

	go s.processKeygen(s.BridgeScanner.KeygenCh)

	go s.signTransactions()

	go s.BridgeScanner.Start()

	for _, v := range s.blockScanners {
		go v.Start()
	}

	return nil
}

func (s *Signer) processTxnOut() {

}

func (s *Signer) processKeygen(ch chan *types.KeygenBlock) {
	for {
		select {
		case <-s.stopCh:
			return
		case keygenBlock := <-ch:
			/*if !more {*/
			/*return*/
			/*}*/
			/*s.logger.Info().Msgf("Received a keygen block %+v from the Thorchain", keygenBlock)*/
			for _, keygenReq := range keygenBlock.Keygens {
				/*// Add pubkeys to pubkey manager for monitoring...*/
				/*// each member might become a yggdrasil pool*/
				/*for _, pk := range keygenReq.GetMembers() {*/
				/*s.pubkeyMgr.AddPubKey(pk, false)*/
				/*}*/
				keygenStart := time.Now()
				pubKey, blame, err := s.tssKeygen.GenerateNewKey(keygenBlock.Height, keygenReq.GetMembers())
				if blame.FailReason == "" {
					/*err := fmt.Errorf("reason: %s, nodes %+v", blame.FailReason, blame.BlameNodes)*/
					/*s.logger.Error().Err(err).Msg("Blame")*/
					fmt.Println("blame is not null, blame reason:", blame.FailReason)
				}
				keygenTime := time.Since(keygenStart).Milliseconds()
				if err != nil {
					/*s.errCounter.WithLabelValues("fail_to_keygen_pubkey", "").Inc()*/
					/*s.logger.Error().Err(err).Msg("fail to generate new pubkey")*/
					fmt.Println(err)
				}
				/*if pubKey.Secp256K1 != "" {*/

				/*}*/

				/*if !pubKey.Secp256k1.IsEmpty() {*/
				/*s.pubkeyMgr.AddPubKey(pubKey.Secp256k1, true)*/
				/*}*/

				if err := s.sendKeygenToBridgeNetwork(keygenBlock.Height, pubKey.Secp256K1, blame, keygenReq.GetMembers(), keygenReq.Type, keygenTime); err != nil {
					/*s.errCounter.WithLabelValues("fail_to_broadcast_keygen", "").Inc()*/
					/*s.logger.Error().Err(err).Msg("fail to broadcast keygen")*/
					fmt.Println(err)
				}

			}
		}
	}
}

func (s *Signer) sendKeygenToBridgeNetwork(height int64, poolPk string, blame types.Blame, input []string, keygenType int32, keygenTime int64) error {
	return nil
}

func (s *Signer) signTransactions() {

}
