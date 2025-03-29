package video

import (
	"encoding/json"
	"log"
	"net/http"

	"github.com/gorilla/websocket"
)

var AllRooms RoomMap

type response struct {
	RoomID string `json:"room_id"`
}

type BroadcastMessage struct {
	Message map[string]interface{}
	RoomID  string
	Client  *websocket.Conn
}

var broadcast = make(chan BroadcastMessage)

func broadcaster() {
	for {
		msg := <-broadcast
		for _, client := range AllRooms.Map[msg.RoomID] {
			if client.Conn != msg.Client {
				client.Mutex.Lock()
				err := client.Conn.WriteJSON(msg.Message)
				client.Mutex.Unlock()

				if err != nil {
					log.Println("Error writing JSON:", err)
					client.Conn.Close()
				}
			}
		}
	}
}

var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool {
		return true
	},
}

func CreateRoomRequestHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Access-Control-Allow-Origin", "*")

	roomID := AllRooms.CreateRoom()

	json.NewEncoder(w).Encode(response{RoomID: roomID})
}

func JoinRoomRequestHandler(w http.ResponseWriter, r *http.Request) {
	query, ok := r.URL.Query()["roomID"]

	if !ok {
		log.Println("roomID is missing, unable to join the call")
		http.Error(w, "roomID is required", http.StatusBadRequest)
		return
	}

	roomID := query[0]

	ws, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Println("Unable to upgrade http to websocket:", err)
		http.Error(w, "Failed to upgrade to WebSocket", http.StatusInternalServerError)
		return
	}

	AllRooms.InsertIntoRoom(roomID, false, ws)

	go broadcaster()

	for {
		var msg BroadcastMessage

		err := ws.ReadJSON(&msg.Message)
		if err != nil {
			log.Println("Error reading JSON:", err)
			ws.Close()
			break
		}

		msg.Client = ws
		msg.RoomID = roomID

		broadcast <- msg
	}
}
