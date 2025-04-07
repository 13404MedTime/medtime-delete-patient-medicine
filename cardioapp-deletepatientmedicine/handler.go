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

// NewRequestBody's Data (map) field will be in this structure
//.   fields
// objects_ids []string
// table_slug string
// object_data map[string]interface
// method string
// app_id string

// but all field will be an interface, you must do type assertion

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

// GetListClientApiResponse This is get list api response
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
	// Send(string(req))

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

	// you may change table slug  it's related your business logic
	var (
		tableSlug               = "patient_medication"
		medicineTakingTableSlug = "medicine_taking"
	)
	// get medicine taking object
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

	//get list objects response example
	getListObjectRequest := Request{
		// some filters
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

	response.Data = map[string]interface{}{}
	response.Status = "done" //if all will be ok else "error"
	responseByte, _ := json.Marshal(response)

	// Send("response - > " + string(responseByte))
	return string(responseByte)
}

func DeleteObjectMany2Many(url, appId string, request RequestMany2Many) (error, Response) {
	response := Response{}

	_, err := DoRequest(url+"/v1/many-to-many/?project-id=a4dc1f1c-d20f-4c1a-abf5-b819076604bc", "DELETE", request, appId)
	if err != nil {
		response.Data = map[string]interface{}{"message": "Error while updating object"}
		response.Status = "error"
		return errors.New("error"), response
	}
	return nil, response
}

func GetListObject(url, tableSlug, appId string, request Request) (GetListClientApiResponse, error, Response) {
	response := Response{}

	getListResponseInByte, err := DoRequest(url+"/v1/object/get-list/"+tableSlug+"?from-ofs=true", "POST", request, appId)
	if err != nil {
		response.Data = map[string]interface{}{"message": "Error while getting single object"}
		response.Status = "error"
		return GetListClientApiResponse{}, errors.New("error"), response
	}
	var getListObject GetListClientApiResponse
	err = json.Unmarshal(getListResponseInByte, &getListObject)
	if err != nil {
		response.Data = map[string]interface{}{"message": "Error while unmarshalling get list object"}
		response.Status = "error"
		return GetListClientApiResponse{}, errors.New("error"), response
	}
	return getListObject, nil, response
}

func GetObjectSlim(url, tableSlug, appId, guid string) (ClientApiResponse, error, Response) {
	response := Response{}
	var getSingleObject ClientApiResponse
	getSingleResponseInByte, err := DoRequest(url+"/v1/object-slim/"+tableSlug+"/"+guid+"?from-ofs=true", "GET", nil, appId)
	if err != nil {
		response.Data = map[string]interface{}{"message": "Error while getting slim single object"}
		response.Status = "error"
		return ClientApiResponse{}, errors.New("error"), response
	}
	err = json.Unmarshal(getSingleResponseInByte, &getSingleObject)
	if err != nil {
		response.Data = map[string]interface{}{"message": "Error while unmarshalling single object"}
		response.Status = "error"
		return ClientApiResponse{}, errors.New("error"), response
	}
	return getSingleObject, nil, response
}

func GetSingleObject(url, tableSlug, appId, guid string) (ClientApiResponse, error, Response) {
	response := Response{}

	var getSingleObject ClientApiResponse
	getSingleResponseInByte, err := DoRequest(url+"/v1/object/"+tableSlug+"/"+guid+"?from-ofs=true", "GET", nil, appId)
	if err != nil {
		response.Data = map[string]interface{}{"message": "Error while getting single object"}
		response.Status = "error"
		return ClientApiResponse{}, errors.New("error"), response
	}
	err = json.Unmarshal(getSingleResponseInByte, &getSingleObject)
	if err != nil {
		response.Data = map[string]interface{}{"message": "Error while unmarshalling single object"}
		response.Status = "error"
		return ClientApiResponse{}, errors.New("error"), response
	}
	return getSingleObject, nil, response
}

func CreateObject(url, tableSlug, appId string, request Request) (Datas, error, Response) {
	response := Response{}

	var createdObject Datas
	createObjectResponseInByte, err := DoRequest(url+"/v1/object/{table_slug}?from-ofs=true", "POST", request, appId)
	if err != nil {
		response.Data = map[string]interface{}{"message": "Error while creating object"}
		response.Status = "error"
		return Datas{}, errors.New("error"), response
	}
	err = json.Unmarshal(createObjectResponseInByte, &createdObject)
	if err != nil {
		response.Data = map[string]interface{}{"message": "Error while unmarshalling create object object"}
		response.Status = "error"
		return Datas{}, errors.New("error"), response
	}
	return createdObject, nil, response
}

func UpdateObject(url, tableSlug, appId string, request Request) (error, Response) {
	response := Response{}

	_, err := DoRequest(url+"/v1/object/{table_slug}?from-ofs=true", "PUT", request, appId)
	if err != nil {
		response.Data = map[string]interface{}{"message": "Error while updating object"}
		response.Status = "error"
		return errors.New("error"), response
	}
	return nil, response
}

func DeleteObject(url, tableSlug, appId, guid string) (error, Response) {
	response := Response{}

	body, err := DoRequest(url+"/v1/object/"+tableSlug+"/"+guid+"?project-id=a4dc1f1c-d20f-4c1a-abf5-b819076604bc", "DELETE", Request{
		map[string]interface{}{"app_id": appId},
	}, appId)
	if err != nil {
		response.Data = map[string]interface{}{"message": "Error while deleting object", "body": body}
		response.Status = "error"
		return errors.New("error"), response
	}
	return nil, response
}

// func main() {
// 	body := `{
// 		"data": {
// 			"action_type": "BEFORE",
// 			"additional_parameters": [],
// 			"app_id": "P-JV2nVIRUtgyPO5xRNeYll2mT4F5QG4bS",
// 			"environment_id": "dcd76a3d-c71b-4998-9e5c-ab1e783264d0",
// 			"method": "DELETE",
// 			"object_data": {
// 				"company_service_environment_id": "dcd76a3d-c71b-4998-9e5c-ab1e783264d0",
// 				"company_service_project_id": "a4dc1f1c-d20f-4c1a-abf5-b819076604bc",
// 				"id": "24888315-7984-47e7-be98-7a227f619c6a"
// 			},
// 			"object_data_before_update": null,
// 			"object_ids": [
// 				"24888315-7984-47e7-be98-7a227f619c6a"
// 			],
// 			"project_id": "a4dc1f1c-d20f-4c1a-abf5-b819076604bc",
// 			"table_slug": "medicine_taking",
// 			"user_id": "0de8f626-388c-4ea8-8213-aa54c1ad4a5d"
// 		}
// 	}`

// 	fmt.Println("resppppppppp --> ", Handle([]byte(body)))
// }
