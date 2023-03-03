package network

import "gitlab.com/thorchain/tss/go-tss/network/http"

type ChainClient struct {
	blockUrl string
}

type ChainClientConfig struct {
	ChainId         string `mapstructure:"chain_id" `
	ChainHost       string `mapstructure:"chain_host"`
	ChainRPC        string `mapstructure:"chain_rpc"`
	ChainHomeFolder string `mapstructure:"chain_home_folder"`
	SignerName      string `mapstructure:"signer_name"`
	SignerPasswd    string
}

func (c *ChainClient) GetCurrentHeight() (int64, error) {
	return http.GetCurrentHeight(c.blockUrl)
}
