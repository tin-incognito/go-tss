package network

import (
	"bridge/x/bridge/common"
	"bridge/x/bridge/types"
	"fmt"
	"time"

	"gitlab.com/thorchain/tss/go-tss/keygen"
	"gitlab.com/thorchain/tss/go-tss/tss"
)

type Keygen struct {
	server *tss.TssServer
}

func NewTssKeygen(server *tss.TssServer) *Keygen {
	return &Keygen{server: server}
}

func (kg *Keygen) GenerateNewKey(keygenBlockHeight int64, pKeys []string) (types.PubKeySet, types.Blame, error) {
	// No need to do key gen
	if len(pKeys) == 0 {
		return types.PubKeySet{}, types.Blame{}, nil
	}
	var keys []string
	for _, item := range pKeys {
		keys = append(keys, string(item))
	}
	keyGenReq := keygen.Request{
		Keys:        keys,
		Version:     "0",
		BlockHeight: keygenBlockHeight,
	}

	// Use the churn try's block to choose the same leader for every node in an Asgard,
	// since a successful keygen requires every node in the Asgard to take part.
	keyGenReq.BlockHeight = keygenBlockHeight

	ch := make(chan bool, 1)
	defer close(ch)
	timer := time.NewTimer(30 * time.Minute)
	defer timer.Stop()

	var resp keygen.Response
	var err error
	go func() {
		resp, err = kg.server.Keygen(keyGenReq)
		ch <- true
	}()

	select {
	case <-ch:
	// do nothing
	case <-timer.C:
		panic("tss keygen timeout")
	}

	// copy blame to our own struct
	blame := types.Blame{
		FailReason: resp.Blame.FailReason,
		IsUnicast:  resp.Blame.IsUnicast,
		BlameNodes: make([]*types.Node, len(resp.Blame.BlameNodes)),
	}
	for i, n := range resp.Blame.BlameNodes {
		blame.BlameNodes[i].Pubkey = n.Pubkey
		blame.BlameNodes[i].BlameData = n.BlameData
		blame.BlameNodes[i].BlameSignature = n.BlameSignature
	}

	if err != nil {
		// the resp from kg.server.Keygen will not be nil
		if blame.FailReason == "" {
			blame.FailReason = err.Error()
		}
		return types.PubKeySet{}, blame, fmt.Errorf("fail to keygen,err:%w", err)
	}

	cpk, err := common.NewPubKey(resp.PubKey)
	if err != nil {
		return types.PubKeySet{}, blame, fmt.Errorf("fail to create common.PubKey,%w", err)
	}

	return types.PubKeySet{Secp256K1: string(cpk), Ed25519: string(cpk)}, blame, nil
}
