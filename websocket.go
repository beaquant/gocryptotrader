package main

import (
	"log"
	"net/http"
	"time"

	"github.com/gorilla/websocket"
	"github.com/thrasher-/gocryptotrader/common"
)

var WebsocketRoutes = Routes{
	Route{
		"ws",
		"GET",
		"/ws",
		WebsocketClientHandler,
	},
}

type WebsocketClient struct {
	ID       int
	Conn     *websocket.Conn
	LastRecv time.Time
}

type WebsocketEvent struct {
	Event string
	Data  interface{}
}

var WebsocketClientHub []WebsocketClient

func WebsocketClientHandler(w http.ResponseWriter, r *http.Request) {
	upgrader := websocket.Upgrader{
		WriteBufferSize: 1024,
		ReadBufferSize:  1024,
	}

	newClient := WebsocketClient{
		ID: len(WebsocketClientHub),
	}

	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Println(err)
		return
	}

	newClient.Conn = conn
	WebsocketClientHub = append(WebsocketClientHub, newClient)
	log.Println("New websocket client connected.")
}

func DisconnectWebsocketClient(id int, err error) {
	for i := range WebsocketClientHub {
		if WebsocketClientHub[i].ID == id {
			WebsocketClientHub[i].Conn.Close()
			WebsocketClientHub = append(WebsocketClientHub[:i], WebsocketClientHub[i+1:]...)
			log.Printf("Disconnected Websocket client, error: %s", err)
			return
		}
	}
}

func SendWebsocketMessage(id int, evt WebsocketEvent) error {
	data, err := common.JSONEncode(&evt)
	if err != nil {
		return err
	}
	for _, x := range WebsocketClientHub {
		if x.ID == id {
			x.Conn.WriteMessage(websocket.TextMessage, data)
		}
	}
	return nil
}

func BroadcastWebsocketMessage(evt WebsocketEvent) error {
	data, err := common.JSONEncode(&evt)
	if err != nil {
		return err
	}

	for _, x := range WebsocketClientHub {
		x.Conn.WriteMessage(websocket.TextMessage, data)
	}
	return nil
}

func HandleWebsocketEvent(evt WebsocketEvent) {
	switch evt.Event {
	case "ticker":
		log.Println(evt.Data)
	}
}

func WebsocketHandler() {
	for {
		for _, x := range WebsocketClientHub {
			msgType, msg, err := x.Conn.ReadMessage()
			if err != nil {
				DisconnectWebsocketClient(x.ID, err)
				continue
			}

			if msgType != websocket.TextMessage {
				DisconnectWebsocketClient(x.ID, err)
				continue
			}

			resp := WebsocketEvent{}
			err = common.JSONDecode(msg, &resp)
			if err != nil {
				log.Println(err)
				continue
			}
			HandleWebsocketEvent(resp)
		}
	}
}
