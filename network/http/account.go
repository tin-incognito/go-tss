package http

import (
	"encoding/json"
	"io/ioutil"
	"net/http"
	"strconv"
)

type AccountInfoResp struct {
	Type    string `json:"@type"`
	Address string `json:"address"`
	PubKey  struct {
		Type    string `json:"@type"`
		Address string `json:"address"`
	} `json:"pub_key"`
	AccountNumber string `json:"account_number"`
	Sequence      string `json:"sequence"`
}

type AccountInfo struct {
	AccountInfoResp
	AccountNumber int64
	Sequence      int64
}

type GetAccountInfoResp struct {
	Account *AccountInfoResp `json:"account"`
}

func NewAccountInfoByResp(acc *AccountInfoResp) (*AccountInfo, error) {
	accountNumber, err := strconv.ParseInt(acc.AccountNumber, 10, 64)
	if err != nil {
		return nil, err
	}
	sequence, err := strconv.ParseInt(acc.Sequence, 10, 64)
	if err != nil {
		return nil, err
	}
	return &AccountInfo{
		AccountInfoResp: AccountInfoResp{
			Type:    acc.Type,
			Address: acc.Address,
			PubKey:  acc.PubKey,
		},
		AccountNumber: accountNumber,
		Sequence:      sequence,
	}, nil
}

func GetAccountInfo(url, address string) (*AccountInfo, error) {
	url = url + "/cosmos/auth/v1beta1/accounts/" + address
	method := "GET"

	accountInfo := &AccountInfo{}
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
	resp := &GetAccountInfoResp{}
	if err = json.Unmarshal(body, &resp); err != nil {
		return nil, err
	}
	accountInfo, err = NewAccountInfoByResp(resp.Account)
	if err != nil {
		return nil, err
	}

	return accountInfo, nil
}
