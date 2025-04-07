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

// Handle a serverless request
func Handle(req []byte) string {
	var response Response
	var request NewRequestBody
	const urlConst = "https://api.admin.u-code.io"

	err := json.Unmarshal(req, &request)
	if err != nil {
		response.Data = map[string]interface{}{"message": "Error while unmarshalling request"}
		response.Status = "error"
		responseByte, _ := json.Marshal(response)
		return string(responseByte)
	}
	if request.Data["app_id"] == nil {
		response.Data = map[string]interface{}{"message": "App id required"}
		response.Status = "error"
		responseByte, _ := json.Marshal(response)
		return string(responseByte)
	}
	appId := request.Data["app_id"].(string)

	var (
		tableSlug               = "patient_medication"
		medicineTakingTableSlug = "medicine_taking"
	)

	medicineTakingData, err, response := GetObjectSlim(urlConst, medicineTakingTableSlug, appId, request.Data["object_data"].(map[string]interface{})["id"].(string))
	if err != nil {
		response.Data = map[string]interface{}{"failed to get medicine taking object, message:": err.Error()}
		response.Status = "error"
		responseByte, _ := json.Marshal(response)
		return string(responseByte)
	}

	medicineTaking := medicineTakingData.Data.Data.Response
	manyReq := RequestMany2Many{
		IdFrom:    medicineTaking["naznachenie_id"].(string),
		IdTo:      []string{medicineTaking["preparati_id"].(string)},
		TableFrom: "naznachenie",
		TableTo:   "preparati",
	}

	err, _ = DeleteObjectMany2Many(urlConst, appId, manyReq)
	if err != nil {
		response.Data = map[string]interface{}{"message": err.Error()}
		response.Status = "error"
		responseByte, _ := json.Marshal(response)
		return string(responseByte)
	}
}

//get list objects response example
getListObjectRequest := Request{
	Data: map[string]interface{}{
		"medicine_taking_id": request.Data["object_data"].(map[string]interface{})["id"].(string),
	},
}

medicineTakeData, err, response := GetListObject(urlConst, tableSlug, appId, getListObjectRequest)
if err != nil {
	responseByte, _ := json.Marshal(response)
	return string(responseByte)
}

var patientMedicationIds []string

for _, medicineData := range medicineTakeData.Data.Data.Response {
	patientMedicationIds = append(patientMedicationIds, cast.ToString(medicineData["guid"]))
}

var req_data = map[string]interface{}{
	"ids": patientMedicationIds,
}

_, err = DoRequest(urlConst+"/v1/object/patient_medication", "DELETE", req_data, appId)
if err != nil {
	response.Data = map[string]interface{}{"message": "Error while deleting many object"}
	response.Status = "error"
	responseByte, _ := json.Marshal(response)
	return string(responseByte)
}

// ! get notifications
notificationsData, err, response := GetListObject(
	urlConst,
	"notifications",
	appId,
	Request{
		Data: map[string]interface{}{
			"client_id":    medicineTakeData.Data.Data.Response[0]["cleints_id"],
			"preparati_id": medicineTaking["preparati_id"].(string),
		},
	},
)
if err != nil {
	responseByte, _ := json.Marshal(response)
	return string(responseByte)
}

var notification_guid []string

for _, notification := range notificationsData.Data.Data.Response {
	notification_guid = append(notification_guid, cast.ToString(notification["guid"]))
}
var m = map[string]interface{}{
	"ids": notification_guid,
}

_, err = DoRequest(urlConst+"/v1/object/notifications", "DELETE", m, appId)
if err != nil {
	response.Data = map[string]interface{}{"message": "Error while deleting many object"}
	response.Status = "error"
	responseByte, _ := json.Marshal(response)
	return string(responseByte)
}
