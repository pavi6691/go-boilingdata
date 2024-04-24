package service

import (
	"encoding/json"
	"fmt"
	"log"

	"github.com/boilingdata/go-boilingdata/constants"
	"github.com/boilingdata/go-boilingdata/models"
	"github.com/boilingdata/go-boilingdata/wsclient"
	cmap "github.com/orcaman/concurrent-map"
)

type UserService struct {
	Wsc  *wsclient.WSSClient
	Auth Auth
}

var queryServiceMap = cmap.New()

func GetUserService(userName string, password string) UserService {
	qs, ok := queryServiceMap.Get(userName)
	if !ok {
		wsclient := wsclient.NewWSSClient(constants.WssUrl, 0, nil)
		qs = UserService{Wsc: wsclient, Auth: Auth{userName: userName, password: password}}
		queryServiceMap.Set(userName, qs)
	}
	return qs.(UserService)
}

func RemoveUser(userName string) {
	queryServiceMap.Remove(userName)
}

func (s *UserService) Query(payloadMessage []byte) (models.Response, error) {

	// If web socket is closed, in case of timeout/user signout/os intruptions etc
	if s.Wsc.IsWebSocketClosed() {
		idToken, err := s.Auth.AuthenticateUser()
		if err != nil {
			return models.Response{}, fmt.Errorf("Error : " + err.Error())
		}
		header, err := s.Auth.GetSignedWssHeader(idToken)
		if err != nil {
			return models.Response{}, fmt.Errorf("Error Signing wssUrl: " + err.Error())
		}
		s.Wsc.SignedHeader = header
		s.Wsc.Connect()
		if s.Wsc.IsWebSocketClosed() {
			return models.Response{}, fmt.Errorf(s.Wsc.Error)
		}
	}
	var payload models.Payload
	if err := json.Unmarshal(payloadMessage, &payload); err != nil {
		log.Println("error unmarshalling Payload : " + err.Error())
		return models.Response{}, fmt.Errorf("error unmarshalling Payload : " + err.Error())
	}
	s.Wsc.SendMessage(payloadMessage, payload)
	response, err := s.Wsc.GetResponseSync(payload.RequestID)
	if response.Data == nil || err != nil {
		errorMessage := ""
		if err != nil {
			errorMessage = err.Error()
		}
		return models.Response{}, fmt.Errorf("Internal Server Error, could not read messages from websocket -> " + errorMessage)
	}
	return response, nil
}
