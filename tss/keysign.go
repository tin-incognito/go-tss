package tss

import (
	"encoding/base64"
	"errors"
	"fmt"
	"sort"
	"strings"
	"sync"
	"time"

	tsslibcommon "github.com/binance-chain/tss-lib/common"
	"github.com/libp2p/go-libp2p/core/peer"

	"gitlab.com/thorchain/tss/go-tss/blame"
	"gitlab.com/thorchain/tss/go-tss/common"
	"gitlab.com/thorchain/tss/go-tss/conversion"
	"gitlab.com/thorchain/tss/go-tss/keysign"
	"gitlab.com/thorchain/tss/go-tss/messages"
	"gitlab.com/thorchain/tss/go-tss/p2p"
	"gitlab.com/thorchain/tss/go-tss/storage"
)

func (t *TssServer) waitForSignatures(msgID, poolPubKey string, msgsToSign [][]byte, sigChan chan string, tWg *sync.WaitGroup) (keysign.Response, error) {
	// TSS keysign include both form party and keysign itself, thus we wait twice of the timeout
	data, err := t.signatureNotifier.WaitForSignature(msgID, msgsToSign, poolPubKey, t.conf.KeySignTimeout, sigChan, tWg)
	if err != nil {
		return keysign.Response{}, err
	}
	// for gg20, it wrap the signature R,S into ECSignature structure
	if len(data) == 0 {
		return keysign.Response{}, errors.New("keysign failed")
	}

	return t.batchSignatures(data, msgsToSign), nil
}

func (t *TssServer) generateSignature(msgID string, msgsToSign [][]byte, req keysign.Request, threshold int, allParticipants []string, localStateItem storage.KeygenLocalState, blameMgr *blame.Manager, keysignInstance *keysign.TssKeySign, sigChan chan string) (keysign.Response, error) {
	fmt.Printf("Start generate signature for msg ID %v pool pubkey %v\n", msgID, req.PoolPubKey)
	allPeersID, err := conversion.GetPeerIDsFromPubKeys(allParticipants)
	if err != nil {
		panic("invalid block height or public key" + err.Error())
		return keysign.Response{
			Status: common.Fail,
			Blame:  blame.NewBlame(blame.InternalError, []blame.Node{}),
		}, nil
	}
	fmt.Println(201)

	oldJoinParty, err := conversion.VersionLTCheck(req.Version, messages.NEWJOINPARTYVERSION)
	if err != nil {
		return keysign.Response{
			Status: common.Fail,
			Blame:  blame.NewBlame(blame.InternalError, []blame.Node{}),
		}, errors.New("fail to parse the version")
	}
	fmt.Println(202)
	// we use the old join party
	if oldJoinParty {
		fmt.Println(203)
		allParticipants = req.SignerPubKeys
		myPk, err := conversion.GetPubKeyFromPeerID(t.p2pCommunication.GetHost().ID().String())
		if err != nil {
			panic("fail to convert the p2p id(%s) to pubkey, turn to wait for signature" + t.p2pCommunication.GetHost().ID().String() + err.Error())
			return keysign.Response{}, p2p.ErrNotActiveSigner
		}
		fmt.Println(204)
		isSignMember := false
		for _, el := range allParticipants {
			if myPk == el {
				isSignMember = true
				break
			}
		}
		fmt.Println("Is sign member ", isSignMember)
		if !isSignMember {
			panic("we(%s) are not the active signer" + t.p2pCommunication.GetHost().ID().String())
			return keysign.Response{}, p2p.ErrNotActiveSigner
		}
		fmt.Println(206)
	}
	fmt.Println(207)

	joinPartyStartTime := time.Now()
	onlinePeers, leader, errJoinParty := t.joinParty(msgID, req.Version, req.BlockHeight, allParticipants, threshold, sigChan)
	fmt.Println("errJoinParty: ", errJoinParty)
	joinPartyTime := time.Since(joinPartyStartTime)
	if errJoinParty != nil {
		fmt.Println("Can not join party")
		// we received the signature from waiting for signature
		if errors.Is(errJoinParty, p2p.ErrSignReceived) {
			return keysign.Response{}, errJoinParty
		}
		t.tssMetrics.KeysignJoinParty(joinPartyTime, false)
		// this indicate we are processing the leaderness join party
		if leader == "NONE" {
			if onlinePeers == nil {
				panic("error before we start join party")
				t.broadcastKeysignFailure(msgID, allPeersID)
				return keysign.Response{
					Status: common.Fail,
					Blame:  blame.NewBlame(blame.InternalError, []blame.Node{}),
				}, nil
			}

			blameNodes, err := blameMgr.NodeSyncBlame(req.SignerPubKeys, onlinePeers)
			if err != nil {
				fmt.Println("fail to get peers to blame")
			}
			t.broadcastKeysignFailure(msgID, allPeersID)
			// make sure we blame the leader as well
			fmt.Printf("fail to form keysign party with online:%+v", onlinePeers)
			panic("f")
			return keysign.Response{
				Status: common.Fail,
				Blame:  blameNodes,
			}, nil
		}
		fmt.Println(209)

		var blameLeader blame.Blame
		leaderPubKey, err := conversion.GetPubKeyFromPeerID(leader)
		if err != nil {
			panic(fmt.Sprintf("fail to convert the peerID to public key %s, err %v", leader, err))
			blameLeader = blame.NewBlame(blame.TssSyncFail, []blame.Node{})
		} else {
			blameLeader = blame.NewBlame(blame.TssSyncFail, []blame.Node{{leaderPubKey, nil, nil}})
		}
		fmt.Println(210)

		t.broadcastKeysignFailure(msgID, allPeersID)
		// make sure we blame the leader as well
		t.logger.Error().Err(errJoinParty).Msgf("messagesID(%s)fail to form keysign party with online:%v", msgID, onlinePeers)
		return keysign.Response{
			Status: common.Fail,
			Blame:  blameLeader,
		}, nil

	}
	fmt.Println(211)
	t.tssMetrics.KeysignJoinParty(joinPartyTime, true)
	isKeySignMember := false
	for _, el := range onlinePeers {
		if el == t.p2pCommunication.GetHost().ID() {
			isKeySignMember = true
		}
	}
	fmt.Println(212)
	if !isKeySignMember {
		// we are not the keysign member so we quit keysign and waiting for signature
		fmt.Printf("we(%s) are not the active signer", t.p2pCommunication.GetHost().ID().String())
		t.logger.Info().Msgf("we(%s) are not the active signer", t.p2pCommunication.GetHost().ID().String())
		return keysign.Response{}, p2p.ErrNotActiveSigner
	}
	fmt.Println(213)
	parsedPeers := make([]string, len(onlinePeers))
	for i, el := range onlinePeers {
		parsedPeers[i] = el.String()
	}

	fmt.Println(214)
	signers, err := conversion.GetPubKeysFromPeerIDs(parsedPeers)
	if err != nil {
		sigChan <- "signature generated"
		return keysign.Response{
			Status: common.Fail,
			Blame:  blame.Blame{},
		}, nil
	}
	fmt.Println(215)
	signatureData, err := keysignInstance.SignMessage(msgsToSign, localStateItem, signers)
	// the statistic of keygen only care about Tss it self, even if the following http response aborts,
	// it still counted as a successful keygen as the Tss model runs successfully.
	fmt.Println(216)
	if err != nil {
		t.logger.Error().Err(err).Msg("err in keysign")
		sigChan <- "signature generated"
		t.broadcastKeysignFailure(msgID, allPeersID)
		blameNodes := *blameMgr.GetBlame()
		return keysign.Response{
			Status: common.Fail,
			Blame:  blameNodes,
		}, nil
	}
	fmt.Println(217)

	sigChan <- "signature generated"
	// update signature notification
	if err := t.signatureNotifier.BroadcastSignature(msgID, signatureData, allPeersID); err != nil {
		return keysign.Response{}, fmt.Errorf("fail to broadcast signature:%w", err)
	}
	fmt.Println(218)

	return t.batchSignatures(signatureData, msgsToSign), nil
}

func (t *TssServer) updateKeySignResult(result keysign.Response, timeSpent time.Duration) {
	if result.Status == common.Success {
		t.tssMetrics.UpdateKeySign(timeSpent, true)
		return
	}
	t.tssMetrics.UpdateKeySign(timeSpent, false)
}

func (t *TssServer) KeySign(req keysign.Request, id uint64) (keysign.Response, error) {
	fmt.Printf("Perform keysign for ID %v\n", id)
	t.logger.Info().Str("pool pub key", req.PoolPubKey).
		Str("signer pub keys", strings.Join(req.SignerPubKeys, ",")).
		Str("msg", strings.Join(req.Messages, ",")).
		Msg("received keysign request")
	emptyResp := keysign.Response{}
	msgID, err := t.requestToMsgId(req)
	if err != nil {
		return emptyResp, err
	}
	fmt.Printf("Perform keysign for req msg ID %+v\n", msgID)

	keysignInstance := keysign.NewTssKeySign(
		t.p2pCommunication.GetLocalPeerID(),
		t.conf,
		t.p2pCommunication.BroadcastMsgChan,
		t.stopChan,
		msgID,
		t.privateKey,
		t.p2pCommunication,
		t.stateManager,
		len(req.Messages),
	)
	fmt.Printf("id %v keysignInstance tss common %+v\n", id, keysignInstance.GetTssCommonStruct())

	keySignChannels := keysignInstance.GetTssKeySignChannels()
	t.p2pCommunication.SetSubscribe(messages.TSSKeySignMsg, msgID, keySignChannels)
	t.p2pCommunication.SetSubscribe(messages.TSSKeySignVerMsg, msgID, keySignChannels)
	t.p2pCommunication.SetSubscribe(messages.TSSControlMsg, msgID, keySignChannels)
	t.p2pCommunication.SetSubscribe(messages.TSSTaskDone, msgID, keySignChannels)

	defer func() {
		t.p2pCommunication.CancelSubscribe(messages.TSSKeySignMsg, msgID)
		t.p2pCommunication.CancelSubscribe(messages.TSSKeySignVerMsg, msgID)
		t.p2pCommunication.CancelSubscribe(messages.TSSControlMsg, msgID)
		t.p2pCommunication.CancelSubscribe(messages.TSSTaskDone, msgID)

		t.p2pCommunication.ReleaseStream(msgID)
		t.signatureNotifier.ReleaseStream(msgID)
		t.partyCoordinator.ReleaseStream(msgID)
	}()

	localStateItem, err := t.stateManager.GetLocalState(req.PoolPubKey)
	if err != nil {
		return emptyResp, fmt.Errorf("fail to get local keygen state: %w", err)
	}
	fmt.Printf("id %v got Local state %+v\n", id, localStateItem)

	var msgsToSign [][]byte
	for _, val := range req.Messages {
		msgToSign, err := base64.StdEncoding.DecodeString(val)
		if err != nil {
			return keysign.Response{}, fmt.Errorf("fail to decode message(%s): %w", strings.Join(req.Messages, ","), err)
		}
		msgsToSign = append(msgsToSign, msgToSign)
	}
	fmt.Println(14)

	sort.SliceStable(msgsToSign, func(i, j int) bool {
		ma, err := common.MsgToHashInt(msgsToSign[i])
		if err != nil {
			t.logger.Error().Err(err).Msgf("fail to convert the hash value")
		}
		mb, err := common.MsgToHashInt(msgsToSign[j])
		if err != nil {
			t.logger.Error().Err(err).Msgf("fail to convert the hash value")
		}
		if ma.Cmp(mb) == -1 {
			return false
		}
		return true
	})
	fmt.Println(15)

	oldJoinParty, err := conversion.VersionLTCheck(req.Version, messages.NEWJOINPARTYVERSION)
	if err != nil {
		return keysign.Response{
			Status: common.Fail,
			Blame:  blame.NewBlame(blame.InternalError, []blame.Node{}),
		}, errors.New("fail to parse the version")
	}
	fmt.Printf("len signer %v, signer %v old join %v\n", len(req.SignerPubKeys), req.SignerPubKeys, oldJoinParty)

	if len(req.SignerPubKeys) == 0 && oldJoinParty {
		return emptyResp, errors.New("empty signer pub keys")
	}
	fmt.Println(17)

	threshold, err := conversion.GetThreshold(len(localStateItem.ParticipantKeys))
	if err != nil {
		t.logger.Error().Err(err).Msg("fail to get the threshold")
		return emptyResp, errors.New("fail to get threshold")
	}
	fmt.Println("threshold ", threshold)
	if len(req.SignerPubKeys) <= threshold && oldJoinParty {
		t.logger.Error().Msgf("not enough signers, threshold=%d and signers=%d", threshold, len(req.SignerPubKeys))
		return emptyResp, errors.New("not enough signers")
	}
	fmt.Println(19)

	blameMgr := keysignInstance.GetTssCommonStruct().GetBlameMgr()

	var receivedSig, generatedSig keysign.Response
	var errWait, errGen error
	sigChan := make(chan string, 2)
	wg := sync.WaitGroup{}
	wg.Add(2)
	keysignStartTime := time.Now()

	tWg := &sync.WaitGroup{}
	tWg.Add(1)

	go func(tWg *sync.WaitGroup) {
		fmt.Println("Start wait for signatures")
		defer wg.Done()
		receivedSig, errWait = t.waitForSignatures(msgID, req.PoolPubKey, msgsToSign, sigChan, tWg)
		fmt.Println("receivedSig:", receivedSig)
		fmt.Println("errWait:", errWait)
		// we received an valid signature indeed
		if errWait == nil {
			sigChan <- "signature received"
			fmt.Printf("for message %s we get the signature from the peer\n", msgID)
			t.logger.Log().Msgf("for message %s we get the signature from the peer", msgID)
			return
		}
		t.logger.Log().Msgf("we fail to get the valid signature with error %v", errWait)
	}(tWg)
	fmt.Println(100)

	// we generate the signature ourselves
	go func(tWg *sync.WaitGroup) {
		// tWg.Wait()
		fmt.Println("Start generateSignature")
		defer wg.Done()
		generatedSig, errGen = t.generateSignature(msgID, msgsToSign, req, threshold, localStateItem.ParticipantKeys, localStateItem, blameMgr, keysignInstance, sigChan)
	}(tWg)
	fmt.Println(101)
	wg.Wait()
	close(sigChan)
	keysignTime := time.Since(keysignStartTime)
	// we received the generated verified signature, so we return
	if errWait == nil {
		t.updateKeySignResult(receivedSig, keysignTime)
		return receivedSig, nil
	}
	fmt.Println(20)
	// for this round, we are not the active signer
	if errors.Is(errGen, p2p.ErrSignReceived) || errors.Is(errGen, p2p.ErrNotActiveSigner) {
		t.updateKeySignResult(receivedSig, keysignTime)
		return receivedSig, nil
	}
	fmt.Println(21)
	// we get the signature from our tss keysign
	t.updateKeySignResult(generatedSig, keysignTime)
	return generatedSig, errGen
}

func (t *TssServer) broadcastKeysignFailure(messageID string, peers []peer.ID) {
	if err := t.signatureNotifier.BroadcastFailed(messageID, peers); err != nil {
		t.logger.Err(err).Msg("fail to broadcast keysign failure")
	}
}

func (t *TssServer) isPartOfKeysignParty(parties []string) bool {
	for _, item := range parties {
		if t.localNodePubKey == item {
			return true
		}
	}
	return false
}

func (t *TssServer) batchSignatures(sigs []*tsslibcommon.ECSignature, msgsToSign [][]byte) keysign.Response {
	var signatures []keysign.Signature
	for i, sig := range sigs {
		msg := base64.StdEncoding.EncodeToString(msgsToSign[i])
		r := base64.StdEncoding.EncodeToString(sig.R)
		s := base64.StdEncoding.EncodeToString(sig.S)
		recovery := base64.StdEncoding.EncodeToString(sig.SignatureRecovery)

		signature := keysign.NewSignature(msg, r, s, recovery)
		signatures = append(signatures, signature)
	}
	return keysign.NewResponse(
		signatures,
		common.Success,
		blame.Blame{},
	)
}
