package remdeshid

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"

	"github.com/gorilla/websocket"
)

// upgrader is used to upgrade HTTP connections to WebSocket connections
var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
	CheckOrigin: func(r *http.Request) bool {
		return true
	},
}

// HIDWebSocketHandler handles incoming WebSocket connections for HID commands
func (c *Controller) HIDWebSocketHandler(w http.ResponseWriter, r *http.Request) {
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Println("Failed to upgrade to websocket:", err)
		return
	}
	defer conn.Close()

	for {
		_, message, err := conn.ReadMessage()
		if err != nil {
			log.Println("Error reading message:", err)
			break
		}
		//log.Printf("Received: %s", message)

		//Try parsing the message as a HIDCommand
		var hidCmd HIDCommand
		if err := json.Unmarshal(message, &hidCmd); err != nil {
			log.Println("Error parsing message:", err)
			continue
		}

		bytes, err := c.ConstructAndSendCmd(&hidCmd)
		if err != nil {
			errmsg := map[string]string{"error": err.Error()}
			if err := conn.WriteJSON(errmsg); err != nil {
				log.Println("Error writing message:", err)
				continue
			}
			log.Println("Error sending command:", err)
			continue
		}

		prettyBytes := ""
		for _, b := range bytes {
			prettyBytes += fmt.Sprintf("0x%02X ", b)
		}
		if err := conn.WriteMessage(websocket.TextMessage, []byte(prettyBytes)); err != nil {
			log.Println("Error writing message:", err)
			continue
		}

	}
}
