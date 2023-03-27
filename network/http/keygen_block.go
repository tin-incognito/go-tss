package http

import (
	"bridge/x/bridge/types"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"strconv"
)

type GetCurrentHeightResp struct {
	Result struct {
		Response struct {
			Data             string `json:"data"`
			LastBlockHeight  string `json:"last_block_height"`
			LastBlockAppHash string `json:"last_block_app_hash"`
		} `json:"response"`
	} `json:"result"`
}

func GetCurrentHeight(url string) (int64, error) {
	url = url + "/abci_info"
	method := "GET"

	client := &http.Client{}
	req, err := http.NewRequest(method, url, nil)

	if err != nil {
		return -1, err
	}
	res, err := client.Do(req)
	if err != nil {
		return -1, err
	}
	defer res.Body.Close()

	body, err := ioutil.ReadAll(res.Body)
	if err != nil {
		return -1, err
	}
	resp := &GetCurrentHeightResp{}
	if err = json.Unmarshal(body, &resp); err != nil {
		return -1, err
	}
	blockHeight, err := strconv.ParseInt(resp.Result.Response.LastBlockHeight, 10, 64)
	if err != nil {
		return -1, err
	}
	return blockHeight, nil
}

type GetKeygenBlockResp struct {
	Code        int `json:"code"`
	KeygenBlock struct {
		Index   string `json:"index"`
		Height  string `json:"height"`
		Keygens []struct {
			Id      string   `json:"id"`
			Type    int      `json:"type"`
			Members []string `json:"members"`
		} `json:"keygens"`
	} `json:"keygenBlock"`
}

var ErrNotFoundRegisterKeyGen = fmt.Errorf("error not found register keygen")
var ErrNotFoundKeyGenBlock = fmt.Errorf("error not found keygen block")
var ErrConnectionRefused = fmt.Errorf("error connection refused")

func GetKeygenBlock(url string, height int64) (*types.KeygenBlock, error) {
	url = url + "/bridge/bridge/keygen_block/%v"
	url = fmt.Sprintf(url, height)
	method := "GET"

	client := &http.Client{}
	req, err := http.NewRequest(method, url, nil)

	if err != nil {
		return nil, err
	}
	res, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer res.Body.Close()

	body, err := ioutil.ReadAll(res.Body)
	if err != nil {
		return nil, err
	}
	resp := &GetKeygenBlockResp{}
	if err = json.Unmarshal(body, &resp); err != nil {
		return nil, err
	}
	if resp.Code != 0 {
		if resp.Code == 5 {
			return nil, ErrNotFoundKeyGenBlock
		} else if resp.Code == 14 {
			return nil, ErrConnectionRefused
		} else {
			return nil, fmt.Errorf("not found keygen block")
		}
	}
	blockHeight, err := strconv.ParseInt(resp.KeygenBlock.Height, 10, 64)
	if err != nil {
		return nil, err
	}
	return &types.KeygenBlock{
		Index:  resp.KeygenBlock.Keygens[0].Id,
		Height: blockHeight,
		Keygens: []*types.KeygenValue{
			{
				Id:      resp.KeygenBlock.Keygens[0].Id,
				Type:    0,
				Members: resp.KeygenBlock.Keygens[0].Members,
			},
		},
	}, nil
}

type GetRegisterKeygenResp struct {
	Code           int `json:"code"`
	RegisterKeygen []struct {
		Index      string   `json:"index"`
		Height     string   `json:"height"`
		Members    []string `json:"members"`
		PoolPubKey string   `json:"pool_pub_key"`
	} `json:"registerKeygen"`
}

func GetRegisterKeygen(url string) (*types.RegisterKeygen, error) {
	url = url + "/bridge/bridge/register_keygen"
	url = fmt.Sprintf(url)
	method := "GET"

	client := &http.Client{}
	req, err := http.NewRequest(method, url, nil)

	if err != nil {
		return nil, err
	}
	res, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer res.Body.Close()

	body, err := ioutil.ReadAll(res.Body)
	if err != nil {
		return nil, err
	}
	resp := &GetRegisterKeygenResp{}
	if err = json.Unmarshal(body, &resp); err != nil {
		return nil, err
	}
	if resp.Code != 0 {
		if resp.Code == 5 {
			return nil, ErrNotFoundKeyGenBlock
		} else if resp.Code == 14 {
			return nil, ErrConnectionRefused
		} else {
			return nil, fmt.Errorf("not found register keygen")
		}
	}
	if len(resp.RegisterKeygen) == 0 {
		return nil, ErrNotFoundRegisterKeyGen
	}
	t := resp.RegisterKeygen[0]
	blockHeight, err := strconv.ParseInt(t.Height, 10, 64)
	if err != nil {
		return nil, err
	}
	return &types.RegisterKeygen{
		Index:      t.Index,
		Height:     blockHeight,
		Members:    t.Members,
		PoolPubKey: t.PoolPubKey,
	}, nil
}
