package api

import (
	"encoding/json"
	"io/ioutil"
	"log"
	"net/http"

	"github.com/pavi6691/boilingdata-sdk-go/service"
	"github.com/pavi6691/boilingdata-sdk-go/wsclient"
)

type Message struct {
	Text string `json:"text"`
}
type Handler struct {
	Wsc *wsclient.WSSClient
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

func (h *Handler) Query(w http.ResponseWriter, r *http.Request) {
	// Check if the request method is POST
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	if !service.IsUserLoggedIn() {
		http.Error(w, "User signed out, Please Login!", http.StatusUnauthorized)
		return
	}

	// If web socket is closed, in case of timeout/user signout/os intruptions etc
	if h.Wsc.IsWebSocketClosed() {
		idToken, err := service.AuthenticateUser("", "")
		if err != nil {
			http.Error(w, "Error : "+err.Error(), http.StatusInternalServerError)
			return
		}
		header, err := service.GetSignedWssHeader(idToken)
		if err != nil {
			http.Error(w, "Error Signing wssUrl: "+err.Error(), http.StatusInternalServerError)
			return
		}
		h.Wsc.SignedHeader = header
		h.Wsc.Connect()
		if h.Wsc.IsWebSocketClosed() {
			http.Error(w, h.Wsc.Error, http.StatusInternalServerError)
			return
		}
	}

	// Read the request body
	body, err := ioutil.ReadAll(r.Body)
	if err != nil {
		http.Error(w, "Failed to read request body", http.StatusInternalServerError)
		return
	}
	// Convert the body bytes to a string
	payload := string(body)
	h.Wsc.SendMessage(payload)
	responseJSON := h.ReadMessage()
	if responseJSON == nil {
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
	}
	// Set response content type to JSON
	w.Header().Set("Content-Type", "application/json")
	// Write JSON response to the response body
	w.Write(responseJSON)

}

func (h *Handler) ReadMessage() []byte {
	var responses []Response
	totMessages := -1
	for {
		_, message, err := h.Wsc.Conn.ReadMessage()
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
