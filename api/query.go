package api

import (
	"io/ioutil"
	"net/http"

	"github.com/pavi6691/boilingdata-sdk-go/service"
	"github.com/pavi6691/boilingdata-sdk-go/wsclient"
)

type Handler struct {
	Wsc     *wsclient.WSSClient
	Service service.Service
}

func (h *Handler) Query(w http.ResponseWriter, r *http.Request) {
	// Check if the request method is POST
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	if !h.Service.IsUserLoggedIn() {
		http.Error(w, "User signed out, Please Login!", http.StatusUnauthorized)
		return
	}
	// Read the request body
	body, err := ioutil.ReadAll(r.Body)
	if err != nil {
		http.Error(w, "Could not read http request body", http.StatusInternalServerError)
		return
	}

	response, err := h.Service.Query(string(body))
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	// Set response content type to JSON
	w.Header().Set("Content-Type", "application/json")
	// Write JSON response to the response body
	w.Write(response)

}
