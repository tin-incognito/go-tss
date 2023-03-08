package network

import "gitlab.com/thorchain/tss/go-tss/network/http"

type ChainClient struct {
	cfg *ChainClientConfig
}

func NewChainClient(cfg *ChainClientConfig) *ChainClient {
	return &ChainClient{
		cfg: cfg,
	}
}

type ChainClientConfig struct {
	ChainId      string `mapstructure:"chain_id" `
	BlockUrl     string `mapstructure:"block_url" `
	Stateurl     string `mapstructure:"state_url" `
	SignerName   string `mapstructure:"signer_name"`
	SignerPasswd string
}

func NewChainClientConfig(chainId, blockUrl, stateUrl, signerName, signerPasswd string) *ChainClientConfig {
	return &ChainClientConfig{
		ChainId:      chainId,
		BlockUrl:     blockUrl,
		Stateurl:     stateUrl,
		SignerName:   signerName,
		SignerPasswd: signerPasswd,
	}
}

func (c *ChainClient) GetCurrentHeight() (int64, error) {
	return http.GetCurrentHeight(c.cfg.BlockUrl)
}
