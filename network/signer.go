package network

import (
	"bridge/x/bridge/types"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"sync"
	"time"

	"gitlab.com/thorchain/tss/go-tss/network/chain"
	"gitlab.com/thorchain/tss/go-tss/tss"
)

type Signer struct {
	wg            *sync.WaitGroup
	stopCh        chan struct{}
	bridgeScanner *BridgeScanner
	bridgeClient  *BridgeClient
	ChainClients  map[int]*ChainClient
	blockScanners map[int]*BlockScanner
	tssKeygen     *Keygen
	tssKeysign    *Keysign

	//logger        zerolog.Logger
}

func NewSigner(tssSever *tss.TssServer, blockUrl, stateUrl string, keys *Keys, cfg *BridgeClientConfig) (*Signer, error) {
	res := &Signer{
		wg:            &sync.WaitGroup{},
		stopCh:        make(chan struct{}),
		bridgeScanner: NewBridgeScanner(blockUrl, stateUrl),
		blockScanners: make(map[int]*BlockScanner),
		tssKeygen:     NewTssKeygen(tssSever),
		tssKeysign:    NewTssKeysign(tssSever),
		bridgeClient:  NewBridgeClient(blockUrl, stateUrl, keys, cfg),
	}
	return res, nil
}

func (s *Signer) Start() error {
	fmt.Println("Start signer")
	go s.processTxnOut()

	go s.processKeygen(s.bridgeScanner.KeygenCh, s.bridgeScanner.RegisterKeygen)

	go s.signTransactions()

	go s.bridgeScanner.Start()

	for _, v := range s.blockScanners {
		go v.Start()
	}

	go s.tssKeysign.Start()

	return nil
}

func (s *Signer) processTxnOut() {

}

func (s *Signer) processKeygen(ch chan *types.KeygenBlock, registerKeygenCh chan *types.RegisterKeygen) {
	for {
		select {
		case <-s.stopCh:
			return
		case registerKeygen := <-registerKeygenCh:
			fmt.Println("Start processing registerKeygen")

			msg := chain.RegisterKeygen{PoolPubKey: registerKeygen.PoolPubKey, Members: registerKeygen.Members}
			data, err := json.Marshal(msg)
			if err != nil {
				panic(err)
			}

			sig, _, err := s.tssKeysign.RemoteSign(data, registerKeygen.PoolPubKey, registerKeygen.Height)
			if err != nil {
				registerKeygenCh <- registerKeygen
				continue
			}

			//TODO: this is the bad way try to improve here
			selfAddress, err := s.bridgeClient.AccountAddress()
			if err != nil {
				panic(err)
			}

			if selfAddress.String() != s.bridgeClient.cfg.RelayerAddress {
				continue
			}
			//

			if err := s.sendRegisterKeygenToBridgeNetwork(data, sig); err != nil {
				/*s.errCounter.WithLabelValues("fail_to_broadcast_keygen", "").Inc()*/
				/*s.logger.Error().Err(err).Msg("fail to broadcast keygen")*/
				panic(err)
			}

		case keygenBlock := <-ch:
			fmt.Println("Start processing keygen block")

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
				if blame.FailReason != "" {
					err := fmt.Errorf("reason: %s, nodes %+v", blame.FailReason, blame.BlameNodes)
					/*s.logger.Error().Err(err).Msg("Blame")*/
					panic(err)
				}
				keygenTime := time.Since(keygenStart).Milliseconds()
				if err != nil {
					/*s.errCounter.WithLabelValues("fail_to_keygen_pubkey", "").Inc()*/
					/*s.logger.Error().Err(err).Msg("fail to generate new pubkey")*/
					panic(err)
				}
				/*if pubKey.Secp256K1 != "" {*/

				/*}*/

				/*if !pubKey.Secp256k1.IsEmpty() {*/
				/*s.pubkeyMgr.AddPubKey(pubKey.Secp256k1, true)*/
				/*}*/

				//TODO: this is the bad way try to improve here
				selfAddress, err := s.bridgeClient.AccountAddress()
				if err != nil {
					panic(err)
				}

				if selfAddress.String() != s.bridgeClient.cfg.RelayerAddress {
					continue
				}
				//

				if err := s.sendKeygenToBridgeNetwork(keygenBlock.Height, pubKey.Secp256K1, blame, keygenReq.GetMembers(), keygenReq.Type, keygenTime); err != nil {
					/*s.errCounter.WithLabelValues("fail_to_broadcast_keygen", "").Inc()*/
					/*s.logger.Error().Err(err).Msg("fail to broadcast keygen")*/
					panic(err)
				}

			}
		}
	}
}

func (s *Signer) sendRegisterKeygenToBridgeNetwork(msg []byte, sig []byte) error {
	encodedSig := base64.StdEncoding.EncodeToString(sig)
	encodedMsg := base64.StdEncoding.EncodeToString(msg)
	selfAddress, err := s.bridgeClient.AccountAddress()
	if err != nil {
		return err
	}
	return s.bridgeClient.sendRegisterKeygenTx(selfAddress.String(), encodedMsg, encodedSig)
}

func (s *Signer) sendKeygenToBridgeNetwork(height int64, poolPk string, blame types.Blame, input []string, keygenType int32, keygenTime int64) error {
	selfAddress, err := s.bridgeClient.AccountAddress()
	if err != nil {
		return err
	}
	return s.bridgeClient.sendKeygenTx(selfAddress.String(), poolPk, &blame, input, keygenType, []string{chain.BridgeChainId}, height, keygenTime)
}

func (s *Signer) signTransactions() {

}
