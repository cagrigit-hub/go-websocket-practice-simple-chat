package handlers

import (
	"fmt"
	"log"
	"net/http"
	"sort"

	"github.com/CloudyKit/jet/v6"
	"github.com/gorilla/websocket"
)

var wsChan = make(chan WsPayload)

var clients = make(map[WebSocketConnection]string)
var views = jet.NewSet(
	jet.NewOSFileSystemLoader("./html"),
	jet.InDevelopmentMode(),
)

var upgradeConnection = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
	CheckOrigin: func(r *http.Request) bool {
		return true
	},
}

func Home(w http.ResponseWriter, r *http.Request) {
	err := renderPage(w, "home.jet", nil)
	if err != nil {
		log.Println(err)
	}

}

// WsJsonResponse defines the response structure for the websocket
type WsJsonResponse struct {
	Action         string   `json:"action"`
	Message        string   `json:"message"`
	MessageType    string   `json:"message_type"`
	ConnectedUsers []string `json:"connected_users"`
}

type WebSocketConnection struct {
	*websocket.Conn
}

type WsPayload struct {
	Action     string              `json:"action"`
	Username   string              `json:"username"`
	Message    string              `json:"message"`
	CurrentCon WebSocketConnection `json:"-"`
}

// WsEndpoint handles the websocket connection
func WsEndpoint(w http.ResponseWriter, r *http.Request) {
	ws, err := upgradeConnection.Upgrade(w, r, nil)
	if err != nil {
		log.Println(err)
	}
	log.Println("Client Successfully Connected")

	var response WsJsonResponse
	response.Message = `<em><small>Connected to the server</small></em>`
	conn := WebSocketConnection{Conn: ws}
	clients[conn] = ""

	err = ws.WriteJSON(response)
	if err != nil {
		log.Println(err)
	}
	go ListenForWs(&conn)

}

func ListenForWs(conn *WebSocketConnection) {
	defer func() {
		if r := recover(); r != nil {
			log.Println("Error in ListenForWs", fmt.Sprintf("%v", r))
		}
	}()
	var payload WsPayload

	for {
		err := conn.ReadJSON(&payload)
		if err != nil {
		} else {
			payload.CurrentCon = *conn
			wsChan <- payload
		}
	}
}

func ListenToWsChannel() {
	var response WsJsonResponse
	for {
		e := <-wsChan
		switch e.Action {
		case "set_username":
			// get a list of all users and send it bak via broadcast
			clients[e.CurrentCon] = e.Username
			users := getUserList()
			response.Action = "list_users"
			response.ConnectedUsers = users
			broadcastToAll(response)
		case "left":
			response.Action = "list-users"
			delete(clients, e.CurrentCon)
			users := getUserList()
			response.ConnectedUsers = users
			broadcastToAll(response)
		}

	}
}

func getUserList() []string {
	var users []string
	for _, v := range clients {
		if v != "" {
			users = append(users, v)
		}
	}
	sort.Strings(users)
	return users
}

func broadcastToAll(msg WsJsonResponse) {
	for client := range clients {
		err := client.WriteJSON(msg)
		if err != nil {
			log.Printf("error: %v", err)
			_ = client.Close()
			delete(clients, client)
		}
	}
}

func renderPage(w http.ResponseWriter, tmpl string, data jet.VarMap) error {
	t, err := views.GetTemplate(tmpl)
	if err != nil {
		log.Println(err)
		return err
	}

	err = t.Execute(w, data, nil)
	if err != nil {
		return err
	}

	return nil
}
