package http

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"strings"
)

func GetBeaconInstructions(url string, height uint64) ([][]string, error) {
	//url = "https://lb-fullnode.incognito.org/fullnode"
	method := "POST"

	type Request struct {
		JsonRPC string        `json:"jsonrpc"`
		Id      int           `json:"id"`
		Method  string        `json:"method"`
		Params  []interface{} `json:"params"`
	}

	temp := []interface{}{}
	temp = append(temp, height)
	temp = append(temp, "2")

	request := &Request{
		JsonRPC: "1.0",
		Id:      1,
		Method:  "retrievebeaconblockbyheight",
		Params:  temp,
	}

	data, err := json.Marshal(request)
	if err != nil {
		return nil, err
	}

	payload := strings.NewReader(string(data))

	client := &http.Client{}

	req, err := http.NewRequest(method, url, payload)
	if err != nil {
		return nil, err
	}

	req.Header.Add("Content-Type", "application/json")

	res, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer res.Body.Close()

	body, err := ioutil.ReadAll(res.Body)
	if err != nil {
		return nil, err
	}

	type Result struct {
		Instructions [][]string `json:"Instructions"`
	}

	type Response struct {
		Result []Result  `json:"Result"`
		Error  *struct{} `json:"Error"`
	}

	response := Response{}

	if err = json.Unmarshal(body, &response); err != nil {
		return nil, err
	}

	if response.Error != nil {
		return nil, fmt.Errorf("can not found beacon height %v", height)
	}

	return response.Result[0].Instructions, nil
}

func GetCurrentBeaconHeight(url string) (uint64, error) {
	//url = "https://lb-fullnode.incognito.org/fullnode"
	method := "POST"

	type Request struct {
		JsonRPC string        `json:"jsonrpc"`
		Id      int           `json:"id"`
		Method  string        `json:"method"`
		Params  []interface{} `json:"params"`
	}

	request := &Request{
		JsonRPC: "1.0",
		Id:      1,
		Method:  "getbeaconbeststate",
		Params:  []interface{}{},
	}

	data, err := json.Marshal(request)
	if err != nil {
		return 0, err
	}

	payload := strings.NewReader(string(data))

	client := &http.Client{}

	req, err := http.NewRequest(method, url, payload)
	if err != nil {
		return 0, err
	}

	req.Header.Add("Content-Type", "application/json")

	res, err := client.Do(req)
	if err != nil {
		return 0, err
	}
	defer res.Body.Close()

	body, err := ioutil.ReadAll(res.Body)
	if err != nil {
		return 0, err
	}

	type Result struct {
		BeaconHeight uint64 `json:"Instructions"`
	}

	type Response struct {
		Result Result    `json:"Result"`
		Error  *struct{} `json:"Error"`
	}

	response := Response{}

	if err = json.Unmarshal(body, &response); err != nil {
		return 0, err
	}

	return response.Result.BeaconHeight, nil
}
