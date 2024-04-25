package wsclient

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"sync"
	"time"

	"github.com/boilingdata/go-boilingdata/constants"
	"github.com/boilingdata/go-boilingdata/models"
	"github.com/gorilla/websocket"
	cmap "github.com/orcaman/concurrent-map"
)

// Use channels to communicate between goroutines
var queryMessageChannel = make(chan []byte)

// each request a response is stored here
var resultsMap = cmap.New()

// To handle async message sharing. need to stop asyn threads in case of error
var isEveythingOK = true

// WSSClient represents the WebSocket client.
type WSSClient struct {
	URL                string
	Conn               *websocket.Conn
	DialOpts           *websocket.Dialer
	idleTimeoutMinutes time.Duration
	idleTimer          *time.Timer
	Wg                 sync.WaitGroup
	ConnInit           sync.WaitGroup
	SignedHeader       http.Header
	Error              string
	mu                 sync.Mutex
}

// NewWSSClient creates a new instance of WSSClient.
// Either fully signed url needs to be provided OR signedHeader
func NewWSSClient(url string, idleTimeoutMinutes time.Duration, signedHeader http.Header) *WSSClient {
	if signedHeader == nil {
		signedHeader = make(http.Header)
	}
	return &WSSClient{
		URL:                url,
		DialOpts:           &websocket.Dialer{},
		idleTimeoutMinutes: idleTimeoutMinutes,
		SignedHeader:       signedHeader,
	}
}

func (wsc *WSSClient) Connect() {
	wsc.mu.Lock()
	if wsc.IsWebSocketClosed() {
		log.Println("Connecting to web socket..")
		wsc.ConnInit.Add(1)
		go wsc.connect()
		wsc.ConnInit.Wait()
		if !wsc.IsWebSocketClosed() {
			log.Println("Websocket Connected!")
		}
	}
	wsc.mu.Unlock()
}

func (wsc *WSSClient) connect() {
	interrupt := make(chan os.Signal, 1)
	signal.Notify(interrupt, os.Interrupt)

	wsc.Wg = sync.WaitGroup{}
	wsc.Wg.Add(1)

	go func() {
		defer wsc.Wg.Done()
		for {
			select {
			case <-interrupt:
				log.Println("Interrupt signal received, closing connection")
				wsc.Close()
				return
			}
		}
	}()
	// Connect to WebSocket server
	conn, _, err := websocket.DefaultDialer.Dial(wsc.URL, wsc.SignedHeader)
	if err != nil {
		wsc.Error = err.Error()
		log.Println("dial:", err)
		wsc.ConnInit.Done()
		return
	}
	wsc.Conn = conn // Assign the connection to the Conn field
	if wsc.idleTimeoutMinutes <= 0 {
		wsc.idleTimeoutMinutes = constants.IdleTimeoutMinutes
	} else {
		wsc.idleTimeoutMinutes = wsc.idleTimeoutMinutes * time.Minute
	}
	isEveythingOK = true
	go wsc.sendMessageAsync()
	go wsc.receiveMessageAsync()
	wsc.resetIdleTimer()
	wsc.ConnInit.Done()
}

// SendMessage sends a message over the WebSocket connection.
func (wsc *WSSClient) SendMessage(message []byte, payload models.Payload) {
	resultsMap.Set("error", nil)
	resultsMap.Set(payload.RequestID, nil)
	queryMessageChannel <- message
}

// Close closes the WebSocket connection. perform clean up
func (wsc *WSSClient) Close() {
	wsc.mu.Lock()
	if wsc.Conn != nil {
		isEveythingOK = false
		wsc.Conn.Close()
		wsc.Conn = nil
		wsc.idleTimer = nil
		resultsMap.Clear()
		log.Println("Websocket connnection closed")
	}
	wsc.mu.Unlock()
}

func (wsc *WSSClient) IsWebSocketClosed() bool {
	return wsc.Conn == nil || !isEveythingOK
}

// resetIdleTimer resets the idle timer.
func (wsc *WSSClient) resetIdleTimer() {
	if wsc.idleTimer != nil {
		wsc.idleTimer.Stop()
	}
	wsc.idleTimer = time.AfterFunc(wsc.idleTimeoutMinutes, func() {
		log.Println("Idle timeout reached, closing connection")
		wsc.Close()
	})
}

// Async function to send message through channel
func (wsc *WSSClient) sendMessageAsync() {
	defer wsc.Close()
	for {
		// Read message from the query message channel
		message, ok := <-queryMessageChannel
		if !ok || !isEveythingOK {
			log.Println("SendMessageAsync process interrupted. No messages will be sent to websocket now onwards.  Action : Reconnect websocket")
			break
		}
		if wsc.Conn == nil {
			wsc.Error = "Could not send message to websocket -> Not connected to WebSocket server"
			resultsMap.Set("error", fmt.Errorf("Could not send message to websocket -> "+"Not connected to WebSocket server"))
			return
		}
		wsc.idleTimer.Reset(constants.IdleTimeoutMinutes)
		wsc.mu.Lock()
		err := wsc.Conn.WriteMessage(websocket.TextMessage, message)
		if err != nil {
			isEveythingOK = false
			resultsMap.Set("error", fmt.Errorf("Could not send message to websocket -> "+"Not connected to WebSocket server"))
		}
		wsc.mu.Unlock()
	}
}

// Async function to receive message through channel
func (wsc *WSSClient) receiveMessageAsync() {
	defer wsc.Close()
	for {
		if wsc.Conn == nil {
			wsc.Error = "Could not receive message from websocket -> Not connected to WebSocket server"
			resultsMap.Set("error", fmt.Errorf("Could not recieve message from websocket -> "+"Not connected to WebSocket server"))
			return
		}
		if !isEveythingOK {
			log.Println("ReceiveMessageAsync process intrrupted. No message will be consumed further. Action : Reconnect websocket")
			break
		}
		_, message, err := wsc.Conn.ReadMessage()
		if err != nil {
			isEveythingOK = false
			resultsMap.Set("error", fmt.Errorf("Could not read message from websocket -> ", err.Error()))
		} else if message != nil {
			var response models.Response
			err = json.Unmarshal([]byte(message), &response)
			if err != nil {
				log.Println("Error parsing JSON:", err)
				resultsMap.Set(response.RequestID, fmt.Errorf("Error parsing JSON: "+err.Error()))
			}
			if v, ok := resultsMap.Get(response.RequestID); !ok || v == nil {
				var responses = make(map[int]models.Response)
				resultsMap.Set(response.RequestID, responses)
			}
			v, _ := resultsMap.Get(response.RequestID)
			v.(map[int]models.Response)[response.SubBatchSerial] = response
		}
	}
}

func (wsc *WSSClient) GetResponseSync(requestID string) (models.Response, error) {
	var temp *models.Response
	for {
		if v, ok := resultsMap.Get("error"); ok {
			if v != nil {
				return models.Response{}, v.(error)
			}
		}
		if _, ok := resultsMap.Get(requestID); !ok {
			continue
		}
		responses, _ := resultsMap.Get(requestID)
		if v, ok := responses.(error); ok {
			return models.Response{}, v
		}
		if responses == nil {
			continue
		}
		if v, ok := responses.(map[int]models.Response); ok {
			if len(v) > 0 {
				if temp == nil {
					for _, res := range v {
						temp = &res
						break
					}
				}
				if temp.TotalBatches <= 0 {
					return models.Response{}, fmt.Errorf("No response from server. Check SQL syntax")
				} else if temp.TotalSubBatches == 0 || temp.TotalSubBatches == len(v) {
					var data []map[string]interface{}
					for i := 0; i <= len(v); i++ {
						data = append(data, v[i].Data...)
					}
					if len(v) > 0 {
						finalResponse := v[len(v)]
						finalResponse.Data = data
						return finalResponse, nil
					}
				}
			}
		}
	}
}
