package service

import (
	"encoding/json"
	"fmt"
	"log"

	"github.com/boilingdata/go-boilingdata/wsclient"
)

type QueryService struct {
	Wsc  *wsclient.WSSClient
	Auth Auth
}

type Response struct {
	MessageType       string                   `json:"messageType"`
	RequestID         string                   `json:"requestId"`
	BatchSerial       int                      `json:"batchSerial"`
	TotalBatches      int                      `json:"totalBatches"`
	SplitSerial       int                      `json:"splitSerial"`
	TotalSplitSerials int                      `json:"totalSplitSerials"`
	CacheInfo         string                   `json:"cacheInfo"`
	SubBatchSerial    int                      `json:"subBatchSerial"`
	TotalSubBatches   int                      `json:"totalSubBatches"`
	Data              []map[string]interface{} `json:"data"`
}

type Payload struct {
	MessageType string `json:"messageType"`
	SQL         string `json:"sql"`
	RequestID   string `json:"requestId"`
	ReadCache   string `json:"readCache"`
	Tags        []Tag  `json:"tags"`
}

// Define structs to represent the JSON payload
type Tag struct {
	Name  string `json:"name"`
	Value string `json:"value"`
}

func (s *QueryService) Query(query string) ([]Response, error) {

	// If web socket is closed, in case of timeout/user signout/os intruptions etc
	if s.Wsc.IsWebSocketClosed() {
		idToken, err := s.Auth.AuthenticateUser()
		if err != nil {
			return []Response{}, fmt.Errorf("Error : " + err.Error())
		}
		header, err := s.Auth.GetSignedWssHeader(idToken)
		if err != nil {
			return []Response{}, fmt.Errorf("Error Signing wssUrl: " + err.Error())
		}
		s.Wsc.SignedHeader = header
		s.Wsc.Connect()
		if s.Wsc.IsWebSocketClosed() {
			return []Response{}, fmt.Errorf(s.Wsc.Error)
		}
	}
	s.Wsc.SendMessage(query)
	response, err := s.ReadMessage()
	if response == nil || err != nil {
		errorMessage := ""
		if err != nil {
			errorMessage = err.Error()
		}
		return []Response{}, fmt.Errorf("internal Server Error, could not read message from websocket -> " + errorMessage)
	}
	return response, nil

}

func (s *QueryService) ReadMessage() ([]Response, error) {
	var responses []Response
	totMessages := -1
	for {
		_, message, err := s.Wsc.Conn.ReadMessage()
		if err != nil {
			log.Println("read:", err)
			return []Response{}, err
		}
		var response Response
		err = json.Unmarshal([]byte(message), &response)
		if err != nil {
			log.Println("Error parsing JSON:", err)
			return []Response{}, err
		}
		responses = append(responses, response)
		if response.TotalSubBatches <= 0 {
			break
		} else if totMessages == -1 {
			totMessages = response.TotalSubBatches
		}
		totMessages--
		if totMessages == 0 {
			break
		}
	}
	return responses, nil
}
