package api

import (
	"encoding/json"
	"io/ioutil"
	"log"
	"net/http"

	"github.com/pavi6691/boilingdata-sdk-go/service"
	auth "github.com/pavi6691/boilingdata-sdk-go/service"
)

type WSSPayload struct {
	WssURL string `json:"wssURL"`
}

func (h *Handler) ConnectWSS(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method Not Allowed", http.StatusMethodNotAllowed)
		return
	}

	// Read the request body
	body, err := ioutil.ReadAll(r.Body)
	if err != nil {
		http.Error(w, "Failed to read request body", http.StatusInternalServerError)
		return
	}

	// Define a variable of type Credentials to store the parsed JSON
	var wssPayload WSSPayload
	// Unmarshal the JSON into the Credentials struct
	if err := json.Unmarshal(body, &wssPayload); err != nil {
		log.Fatalf("failed to parse JSON: %v", err)
	}
	headers, err := auth.GetAWSSingingHeaders(wssPayload.WssURL)
	if err != nil {
		http.Error(w, "Error preparing request", http.StatusInternalServerError)
		return
	}
	h.Wsc.SignedHeader = headers
	h.Wsc.Connect()
	if h.Wsc.IsWebSocketClosed() {
		http.Error(w, h.Wsc.Error, http.StatusInternalServerError)
	} else {
		w.Write([]byte("Connected!"))
	}
}

func (h *Handler) GetSignedWSSUrl(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method Not Allowed", http.StatusMethodNotAllowed)
		return
	}

	if !service.IsUserLoggedIn() {
		http.Error(w, "User signed out, Please Login!", http.StatusUnauthorized)
		return
	}

	idToken, err := service.AuthenticateUser("", "")
	if err != nil {
		http.Error(w, "Error : "+err.Error(), http.StatusInternalServerError)
		return
	}

	headers, err := auth.GetSignedWssHeader(idToken)
	if err != nil {
		http.Error(w, "Error getting signed headers: "+err.Error(), http.StatusInternalServerError)
		return
	}
	sigedUrl, err := auth.GetSignedWssUrl(headers)
	if err != nil {
		http.Error(w, "Error Signing wssUrl: "+err.Error(), http.StatusInternalServerError)
		return
	}
	w.Write([]byte(sigedUrl))
}
