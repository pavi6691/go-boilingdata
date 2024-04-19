package service

import (
	"encoding/json"
	"fmt"
	"log"
)

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

func (s *Service) Query(query string) ([]byte, error) {

	// If web socket is closed, in case of timeout/user signout/os intruptions etc
	if s.Wsc.IsWebSocketClosed() {
		idToken, err := s.AuthenticateUser("", "")
		if err != nil {
			return nil, fmt.Errorf("Error : " + err.Error())
		}
		header, err := s.GetSignedWssHeader(idToken)
		if err != nil {
			return nil, fmt.Errorf("Error Signing wssUrl: " + err.Error())
		}
		s.Wsc.SignedHeader = header
		s.Wsc.Connect()
		if s.Wsc.IsWebSocketClosed() {
			return nil, fmt.Errorf(s.Wsc.Error)
		}
	}
	s.Wsc.SendMessage(query)
	responseJSON := s.ReadMessage()
	if responseJSON == nil {
		return nil, fmt.Errorf("internal Server Error, could not read message from websocket")
	}
	return responseJSON, nil

}

func (s *Service) ReadMessage() []byte {
	var responses []Response
	totMessages := -1
	for {
		_, message, err := s.Wsc.Conn.ReadMessage()
		if err != nil {
			log.Println("read:", err)
			break
		}
		var response Response
		err = json.Unmarshal([]byte(message), &response)
		if err != nil {
			log.Println("Error parsing JSON:", err)
			return nil
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
	responseJSON, err := json.MarshalIndent(responses, "", "    ")
	if err != nil {
		return nil
	}
	return responseJSON
}
