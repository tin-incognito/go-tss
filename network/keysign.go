package network

import (
	"encoding/base64"
	"fmt"
	"math/big"
	"math/rand"
	"sort"
	"sync"
	"time"

	"github.com/tendermint/btcd/btcec"
	"gitlab.com/thorchain/tss/go-tss/config"
	"gitlab.com/thorchain/tss/go-tss/keysign"
	"gitlab.com/thorchain/tss/go-tss/tss"
)

const (
	maxKeysignPerRequest = 15 // the maximum number of messages include in one single TSS keysign request
)

type Keysign struct {
	server    *tss.TssServer
	taskQueue chan *TssKeySignTask
	wg        *sync.WaitGroup
	done      chan struct{}
}

func NewTssKeysign(server *tss.TssServer) *Keysign {
	return &Keysign{
		server:    server,
		taskQueue: make(chan *TssKeySignTask),
		done:      make(chan struct{}),
		wg:        &sync.WaitGroup{},
	}
}

type TssKeySignResult struct {
	R          string
	S          string
	RecoveryID string
	Err        error
}

type TssKeySignTask struct {
	ID          uint64
	PoolPubKey  string
	Msg         string
	BlockHeight int64
	Resp        chan TssKeySignResult
}

// Start the keysign workers
func (ks *Keysign) Start() {
	ks.wg.Add(1)
	go ks.processKeySignTasks()

}

func (ks *Keysign) RemoteSign(msg []byte, poolPubKey string, blockHeight int64, signID uint64) ([]byte, []byte, error) {
	c := config.GetConfig()
	if len(msg) == 0 {
		return nil, nil, nil
	}
	encodedMsg := base64.StdEncoding.EncodeToString(msg)
	task := TssKeySignTask{
		PoolPubKey:  poolPubKey,
		Msg:         encodedMsg,
		Resp:        make(chan TssKeySignResult, 1),
		BlockHeight: blockHeight,
		ID:          signID,
	}
	fmt.Printf("SignID: %v Create task %+v and send to queue\n", signID, task)
	ks.taskQueue <- &task
	select {
	case resp := <-task.Resp:
		fmt.Printf("Received tss keysign task response from taskID %v\n", task.ID)
		if resp.Err != nil {
			return nil, nil, fmt.Errorf("fail to tss sign: %w, task %v", resp.Err, task.ID)
		}

		if len(resp.R) == 0 && len(resp.S) == 0 {
			// this means the node tried to do keysign , however this node has not been chosen to take part in the keysign committee
			return nil, nil, nil
		}
		//s.logger.Debug().Str("R", resp.R).Str("S", resp.S).Str("recovery", resp.RecoveryID).Msg("tss result")
		data, err := getSignature(resp.R, resp.S)
		if err != nil {
			return nil, nil, fmt.Errorf("fail to decode tss signature: %w, taskID %v", err, task.ID)
		}
		bRecoveryId, err := base64.StdEncoding.DecodeString(resp.RecoveryID)
		if err != nil {
			return nil, nil, fmt.Errorf("fail to decode recovery id: %w, taskID %v", err, task.ID)
		}
		return data, bRecoveryId, nil
	case <-time.After(c.TssConfig.KeySignTimeout):
		return nil, nil, fmt.Errorf("TIMEOUT: fail to sign message:%s after %d seconds, signID %v", encodedMsg, c.TssConfig.KeySignTimeout/time.Second, signID)
	}
}

func getSignature(r, s string) ([]byte, error) {
	rBytes, err := base64.StdEncoding.DecodeString(r)
	if err != nil {
		return nil, err
	}
	sBytes, err := base64.StdEncoding.DecodeString(s)
	if err != nil {
		return nil, err
	}

	R := new(big.Int).SetBytes(rBytes)
	S := new(big.Int).SetBytes(sBytes)
	N := btcec.S256().N
	halfOrder := new(big.Int).Rsh(N, 1)
	// see: https://github.com/ethereum/go-ethereum/blob/f9401ae011ddf7f8d2d95020b7446c17f8d98dc1/crypto/signature_nocgo.go#L90-L93
	if S.Cmp(halfOrder) == 1 {
		S.Sub(N, S)
	}

	// Serialize signature to R || S.
	// R, S are padded to 32 bytes respectively.
	rBytes = R.Bytes()
	sBytes = S.Bytes()

	sigBytes := make([]byte, 64)
	// 0 pad the byte arrays from the left if they aren't big enough.
	copy(sigBytes[32-len(rBytes):32], rBytes)
	copy(sigBytes[64-len(sBytes):64], sBytes)
	return sigBytes, nil
}

func (ks *Keysign) processKeySignTasks() {
	defer ks.wg.Done()
	tasks := make(map[string][]*TssKeySignTask)
	taskLock := sync.Mutex{}
	for {
		select {
		case <-ks.done:
			// requested to exit
			return
		case t := <-ks.taskQueue:
			fmt.Printf("task %v Enter task queue\n", t.ID)
			taskLock.Lock()
			_, ok := tasks[t.PoolPubKey]
			if !ok {
				tasks[t.PoolPubKey] = []*TssKeySignTask{
					t,
				}
			} else {
				tasks[t.PoolPubKey] = append(tasks[t.PoolPubKey], t)
			}
			taskLock.Unlock()
		case <-time.After(time.Second):
			// This implementation will check the tasks every second , and send whatever is in the queue to TSS
			// if it has more than maxKeysignPerRequest(15) in the queue , it will only send the first maxKeysignPerRequest(15) of them
			// the reset will be send in the next request
			taskLock.Lock()
			xID := rand.Int63()
			fmt.Println("Check KeySignTasks, ID ", xID)
			for k, v := range tasks {
				fmt.Printf("Pool Pubkey %v, value %+v, threadID %v\n", k, v, xID)
				if len(v) == 0 {
					fmt.Printf("Len value is 0, Pool Pubkey %v, value %+v, threadID %v\n", k, v, xID)
					delete(tasks, k)
					continue
				}
				totalTasks := len(v)
				// send no more than maxKeysignPerRequest messages in a single TSS keysign request
				if totalTasks > maxKeysignPerRequest {
					totalTasks = maxKeysignPerRequest
					// when there are more than maxKeysignPerRequest messages in the task queue need to be signed
					// the messages has to be sorted , because the order of messages that get into the slice is not deterministic
					// so it need to sorted to make sure all bifrosts send the same messages to tss
					sort.SliceStable(v, func(i, j int) bool {
						return v[i].Msg < v[j].Msg
					})
				}
				ks.wg.Add(1)
				signingTask := v[:totalTasks]
				tasks[k] = v[totalTasks:]
				fmt.Printf("Perform signing task %v %+v xID %v\n", k, signingTask, xID)
				go ks.toLocalTSSSigner(k, signingTask, uint64(xID))
			}
			taskLock.Unlock()
		}
	}
}

// toLocalTSSSigner will send the request to local signer
func (ks *Keysign) toLocalTSSSigner(poolPubKey string, tasks []*TssKeySignTask, ID uint64) {
	fmt.Printf("xID %v toLocalTSSSigner\n", ID)
	defer ks.wg.Done()
	var msgToSign []string
	var blockHeight int64
	for _, item := range tasks {
		fmt.Printf("Get msg %v from task %v to sign\n", item.Msg, item.ID)
		msgToSign = append(msgToSign, item.Msg)
		blockHeight = item.BlockHeight
	}
	tssMsg := keysign.Request{
		PoolPubKey:  poolPubKey,
		Messages:    msgToSign,
		Version:     "0.14.0",
		BlockHeight: blockHeight,
	}

	//s.logger.Debug().Msg("new TSS join party")
	// get current thorchain block height
	/*blockHeight, err := ks.bridge.GetBlockHeight()*/
	/*if err != nil {*/
	/*s.setTssKeySignTasksFail(tasks, fmt.Errorf("fail to get block height from thorchain: %w", err))*/
	/*return*/
	/*}*/
	/*// this is just round the block height to the nearest 20*/
	/*tssMsg.BlockHeight = blockHeight / 20 * 20*/

	//s.logger.Info().Msgf("msgToSign to tss Local node PoolPubKey: %s, Messages: %+v, block height: %d", tssMsg.PoolPubKey, tssMsg.Messages, tssMsg.BlockHeight)

	keySignResp, err := ks.server.KeySign(tssMsg, ID)
	if err != nil {
		//s.setTssKeySignTasksFail(tasks, fmt.Errorf("fail tss keysign: %w", err))
		fmt.Printf("Can not sign task %v msgs %v, err: %v, xID %v\n", tssMsg.PoolPubKey, tssMsg.Messages, err, ID)
		return
	}

	// 1 means success,2 means fail , 0 means NA
	if keySignResp.Status == 1 && len(keySignResp.Blame.BlameNodes) == 0 {
		//s.logger.Info().Msgf("response: %+v", keySignResp)
		// success
		for _, t := range tasks {
			fmt.Printf("1. ks resp %+v, task %v", keySignResp, t.ID)
			found := false
			for _, sig := range keySignResp.Signatures {
				if t.Msg == sig.Msg {
					t.Resp <- TssKeySignResult{
						R:          sig.R,
						S:          sig.S,
						RecoveryID: sig.RecoveryID,
						Err:        nil,
					}
					found = true
					break
				}
			}
			// Didn't find the signature in the tss keysign result , notify the task , so it doesn't get stuck
			if !found {
				t.Resp <- TssKeySignResult{
					Err: fmt.Errorf("didn't find signature for message %s in the keysign result", t.Msg),
				}
			}
		}
		return
	}
	// fmt.Println(3)

	// copy blame to our own struct
	/*blame := types.Blame{*/
	/*FailReason: keySignResp.Blame.FailReason,*/
	/*IsUnicast:  keySignResp.Blame.IsUnicast,*/
	/*//BlameNodes: make([]types.Node, len(keySignResp.Blame.BlameNodes)),*/
	/*}*/

	// fmt.Println(4)

	//fmt.Println("keySignResp.Blame.BlameNodes:", keySignResp.Blame.BlameNodes)
	/*for i, n := range keySignResp.Blame.BlameNodes {*/
	/*blame.BlameNodes[i].Pubkey = n.Pubkey*/
	/*blame.BlameNodes[i].BlameData = n.BlameData*/
	/*blame.BlameNodes[i].BlameSignature = n.BlameSignature*/
	/*}*/

	fmt.Println(5)

	// Blame need to be passed back to thorchain , so as thorchain can use the information to slash relevant node account
	//TODO: set key sign task fail here
	//s.setTssKeySignTasksFail(tasks, NewKeysignError(blame))
}
