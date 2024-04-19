package api

import (
	"encoding/json"
	"io/ioutil"
	"net/http"

	"github.com/pavi6691/boilingdata-sdk-go/service"
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
	_, err = service.AuthenticateUser(creds.UserName, creds.Password)
	if err != nil {
		http.Error(w, "Error : "+err.Error(), http.StatusInternalServerError)
		return
	}
	w.Write([]byte("Login Successful!"))
}
