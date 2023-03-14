package common

import (
	"time"
)

type BridgeConfig struct {
	BlockUrl     string
	StateUrl     string
	RpcUrl       string
	SignerName   string
	SignerPasswd string // TODO: This is the bad way, try to improve later
	//SignerPrivateKey string // TODO: This is the bad way, try to improve later
	RelayerAddress string // TODO: This is the bad way, try to improve later
}

type ChainConfig struct {
	Url string
}

type TssConfig struct {
	// Party Timeout defines how long do we wait for the party to form
	PartyTimeout time.Duration
	// KeyGenTimeoutSeconds defines how long do we wait the keygen parties to pass messages along
	KeyGenTimeout time.Duration
	// KeySignTimeoutSeconds defines how long do we wait keysign
	KeySignTimeout time.Duration
	// Pre-parameter define the pre-parameter generations timeout
	PreParamTimeout time.Duration
	// enable the tss monitor
	EnableMonitor bool
}
