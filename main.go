package main

import (
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"github.com/gorilla/websocket"
	"strings"
)

const MESSAGE_NEW_USER = "New User"
const MESSAGE_CHAT = "Chat"
const MESSAGE_LEAVE = "Leave"

type SocketPayload struct {
	Message string
}

type SocketResponse struct {
	From    string
	Type    string
	Message string
}

type WebSocketConnection struct {
	*websocket.Conn
	Username string
}

var upgrader = websocket.Upgrader{
	ReadBufferSize:1024,
	WriteBufferSize:1024,
}

var connections = make([]*WebSocketConnection, 0)

func main(){
	setupRoutes()
	log.Fatal(http.ListenAndServe(":8080", nil))
}

func setupRoutes(){
	http.HandleFunc("/", homePage)
	http.HandleFunc("/ws", endPoint)
}

func homePage(w http.ResponseWriter, r *http.Request){
	content, err := ioutil.ReadFile("index.html")
	if err != nil {
		http.Error(w, "Could not open requested file", http.StatusInternalServerError)
		return
	}
	fmt.Fprintf(w, "%s", content)
}

func endPoint(w http.ResponseWriter, r *http.Request){
	con, err := websocket.Upgrade(w, r, w.Header(), 1024, 1024)
	if err != nil {
		log.Print(err)
	}

	username := r.URL.Query().Get("username")
	currentConn := WebSocketConnection{Conn: con, Username:username}
	connections = append(connections, &currentConn)

	go handleIO(&currentConn)
}

func handleIO(currentConn *WebSocketConnection) {

	defer func() {
		if r := recover(); r != nil {
			log.Println("ERROR", fmt.Sprintf("%v", r))
		}
	}()

	broadcastMessage(currentConn, MESSAGE_NEW_USER, "")

	for {
		payload := SocketPayload{}
		err := currentConn.ReadJSON(&payload)
		if err != nil {
			if strings.Contains(err.Error(), "websocket: close") {
				broadcastMessage(currentConn, MESSAGE_LEAVE, "")
				ejectConnection(currentConn)
				return
			}
			log.Println(err)
			continue
		}

		broadcastMessage(currentConn, MESSAGE_CHAT, payload.Message)
	}
}

func ejectConnection(con *WebSocketConnection) {
	li := make([]*WebSocketConnection, 0)
	for _, item := range connections {
		if item == con {
			continue
		}
		li = append(li, item)
	}
	connections = li
}

func broadcastMessage(currentConn *WebSocketConnection, kind, message string) {
	for _, item := range connections {
		if item == currentConn {
			continue
		}

		item.WriteJSON(SocketResponse{
			From:    currentConn.Username,
			Type:    kind,
			Message: message,
		})
	}
}