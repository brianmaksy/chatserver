package handlers

import (
	"fmt"
	"log"
	"net/http"
	"sort"

	"github.com/CloudyKit/jet/v6"
	"github.com/gorilla/websocket"
)

var wsChan = make(chan WsPayload) // only accept type WsPayload

var clients = make(map[WebSocketConnection]string)

var views = jet.NewSet(
	jet.NewOSFileSystemLoader("./html"),
	jet.InDevelopmentMode(),
)

var upgradeConnection = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
	CheckOrigin:     func(r *http.Request) bool { return true },
}

func Home(w http.ResponseWriter, r *http.Request) {
	// to render a jet template.
	err := renderPage(w, "home.jet", nil)
	if err != nil {
		log.Println(err)
	}

}

// WsJsonResponse defines the response sent back from websocket.
type WsJsonResponse struct {
	Action         string   `json:"action"`
	Message        string   `json:"message"`
	MessageType    string   `json:"message_type"`
	ConnectedUsers []string `json:"connected_users"`
}

type WebSocketConnection struct {
	*websocket.Conn
}

// kind of info sending to servor.
type WsPayload struct {
	Action   string              `json:"action"`
	Username string              `json:"username"`
	Message  string              `json:"message"`
	Conn     WebSocketConnection `json:"-"`
}

// WsEndpoint upgrades connection to websocket.
func WsEndpoint(w http.ResponseWriter, r *http.Request) {
	// upgrade connection
	ws, err := upgradeConnection.Upgrade(w, r, nil)
	if err != nil {
		log.Println(err)
	}
	log.Println("Client connected to endpoint")
	// send back response
	var response WsJsonResponse
	response.Message = `<em><small>Connected to server</small></em>`

	conn := WebSocketConnection{Conn: ws}
	clients[conn] = ""

	err = ws.WriteJSON(response)
	if err != nil {
		log.Println(err)
	}

	go ListenForWs(&conn)
}

// a go routine.
func ListenForWs(conn *WebSocketConnection) {
	// if go routine stops, it comes back.
	defer func() {
		if r := recover(); r != nil {
			// e.g. if panic
			log.Println("Error", fmt.Sprintf("%v", r))
		}
	}()
	var payload WsPayload
	// infinite loop.
	for {
		err := conn.ReadJSON(&payload)
		if err != nil {
			// do nothing
		} else {
			payload.Conn = *conn
			wsChan <- payload
		}
	}
}

// make channel do something every time we get payload. Called in main.go.
func ListenToWsChannel() {
	var response WsJsonResponse

	for {
		event := <-wsChan

		switch event.Action {
		case "username":
			// get a list of all users and send it back via broadcast function
			clients[event.Conn] = event.Username
			users := getUserList()
			response.Action = "list_users"
			response.ConnectedUsers = users
			broadcastToAll(response)

		case "left":
			// remove user from list of users
			response.Action = "list_users"
			delete(clients, event.Conn)
			users := getUserList()
			response.ConnectedUsers = users
			broadcastToAll(response)

		case "broadcast":
			response.Action = "broadcast"
			response.Message = fmt.Sprintf("<strong>%s</strong>: %s", event.Username, event.Message)
			broadcastToAll(response)
		}
	}
}

func getUserList() []string {
	var userList []string
	for _, user := range clients {
		if user != "" {
			userList = append(userList, user)
		}
	}
	sort.Strings(userList)
	return userList
}

func broadcastToAll(response WsJsonResponse) {
	for client := range clients {
		err := client.WriteJSON(response)
		if err != nil {
			log.Println("websocket err")
			_ = client.Close()
			delete(clients, client)

		}
	}
}

func renderPage(w http.ResponseWriter, tmpl string, data jet.VarMap) error {
	view, err := views.GetTemplate(tmpl)
	if err != nil {
		log.Println(err)
		return err
	}
	err = view.Execute(w, data, nil) // 3rd argument is context - igonred for now.
	if err != nil {
		log.Println(err)
		return err
	}
	return err
}
