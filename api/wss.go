package api

import (
	"encoding/json"
	"io/ioutil"
	"log"
	"net/http"
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
	headers, err := h.UserService.Auth.GetAWSSingingHeaders(wssPayload.WssURL)
	if err != nil {
		http.Error(w, "Error preparing request", http.StatusInternalServerError)
		return
	}
	h.UserService.Wsc.SignedHeader = headers
	h.UserService.Wsc.Connect()
	if h.UserService.Wsc.IsWebSocketClosed() {
		http.Error(w, h.UserService.Wsc.Error, http.StatusInternalServerError)
	} else {
		w.Write([]byte("Connected!"))
	}
}

func (h *Handler) GetSignedWSSUrl(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method Not Allowed", http.StatusMethodNotAllowed)
		return
	}

	if !h.UserService.Auth.IsUserLoggedIn() {
		http.Error(w, "User signed out, Please Login!", http.StatusUnauthorized)
		return
	}

	idToken, err := h.UserService.Auth.AuthenticateUser()
	if err != nil {
		http.Error(w, "Error : "+err.Error(), http.StatusInternalServerError)
		return
	}

	headers, err := h.UserService.Auth.GetSignedWssHeader(idToken)
	if err != nil {
		http.Error(w, "Error getting signed headers: "+err.Error(), http.StatusInternalServerError)
		return
	}
	sigedUrl, err := h.UserService.Auth.GetSignedWssUrl(headers)
	if err != nil {
		http.Error(w, "Error Signing wssUrl: "+err.Error(), http.StatusInternalServerError)
		return
	}
	w.Write([]byte(sigedUrl))
}
