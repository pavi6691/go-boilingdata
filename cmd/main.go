package main

import (
	"log"
	"net/http"

	"github.com/pavi6691/boilingdata-sdk-go/api"
	"github.com/pavi6691/boilingdata-sdk-go/constants"
	"github.com/pavi6691/boilingdata-sdk-go/service"
	"github.com/pavi6691/boilingdata-sdk-go/wsclient"
)

func main() {
	wsclient := wsclient.NewWSSClient(constants.WssUrl, 0, nil)
	handler := &api.Handler{Wsc: wsclient, QueryService: service.QueryService{Wsc: wsclient, Auth: service.Auth{}}}
	http.HandleFunc("/login", handler.Login)
	http.HandleFunc("/connect", handler.ConnectWSS)
	http.HandleFunc("/query", handler.Query)
	http.HandleFunc("/wssurl", handler.GetSignedWSSUrl)
	log.Println("Server is running on port 8088...")
	http.ListenAndServe(":8088", nil)
}
