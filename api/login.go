package api

import (
	"encoding/json"
	"io/ioutil"
	"net/http"
)

type Credentials struct {
	UserName string `json:"userName"`
	Password string `json:"password"`
}

func (h *Handler) Login(w http.ResponseWriter, r *http.Request) {
	// Check if the request method is POST
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Read the request body
	body, err := ioutil.ReadAll(r.Body)
	if err != nil {
		http.Error(w, "Failed to read request body", http.StatusInternalServerError)
		return
	}

	// Define a variable of type Credentials to store the parsed JSON
	var creds Credentials
	// Unmarshal the JSON into the Credentials struct
	if err := json.Unmarshal(body, &creds); err != nil {
		http.Error(w, "failed to parse JSON: %v", http.StatusInternalServerError)
		return
	}
	h.QueryService.Auth.UserName = creds.UserName
	h.QueryService.Auth.Password = creds.Password
	_, err = h.QueryService.Auth.AuthenticateUser()
	if err != nil {
		http.Error(w, "Error : "+err.Error(), http.StatusInternalServerError)
		return
	}
	w.Write([]byte("Login Successful!"))
}
