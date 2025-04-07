package function

import (
	"bytes"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/url"
	"time"

	"github.com/spf13/cast"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

// Datas This is response struct from create
type Datas struct {
	Data struct {
		Data struct {
			Data map[string]interface{} `json:"data"`
		} `json:"data"`
	} `json:"data"`
}

// ClientApiResponse This is get single api response
type ClientApiResponse struct {
	Data ClientApiData `json:"data"`
}

type ClientApiData struct {
	Data ClientApiResp `json:"data"`
}

type ClientApiResp struct {
	Response map[string]interface{} `json:"response"`
}

type Response struct {
	Status string                 `json:"status"`
	Data   map[string]interface{} `json:"data"`
}

type HttpRequest struct {
	Method  string      `json:"method"`
	Path    string      `json:"path"`
	Headers http.Header `json:"headers"`
	Params  url.Values  `json:"params"`
	Body    []byte      `json:"body"`
}

type AuthData struct {
	Type string                 `json:"type"`
	Data map[string]interface{} `json:"data"`
}

type NewRequestBody struct {
	RequestData HttpRequest            `json:"request_data"`
	Auth        AuthData               `json:"auth"`
	Data        map[string]interface{} `json:"data"`
}

type Request struct {
	Data map[string]interface{} `json:"data"`
}

type GetListClientApiResponse struct {
	Data GetListClientApiData `json:"data"`
}

type GetListClientApiData struct {
	Data GetListClientApiResp `json:"data"`
}

type GetListClientApiResp struct {
	Response []map[string]interface{} `json:"response"`
}

type RequestMany2Many struct {
	IdFrom    string   `json:"id_from"`
	IdTo      []string `json:"id_to"`
	TableFrom string   `json:"table_from"`
	TableTo   string   `json:"table_to"`
}

func DoRequest(url string, method string, body interface{}, appId string) ([]byte, error) {
	data, err := json.Marshal(&body)
	if err != nil {
		return nil, err
	}
	client := &http.Client{
		Timeout: time.Duration(5 * time.Second),
	}

	request, err := http.NewRequest(method, url, bytes.NewBuffer(data))
	if err != nil {
		return nil, err
	}
	request.Header.Add("authorization", "API-KEY")
	request.Header.Add("X-API-KEY", appId)

	resp, err := client.Do(request)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	respByte, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	return respByte, nil
}

func Send(text string) {
	bot, _ := tgbotapi.NewBotAPI("7090962267:AAFR8hJWgTvYS27nVcurE9uLoVzURWxWEZk")

	msg := tgbotapi.NewMessage(162256495, text)

	bot.Send(msg)
}
