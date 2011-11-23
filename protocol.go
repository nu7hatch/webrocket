package webrocket

import (
	"websocket"
	"log"
	"os"
	"fmt"
	"container/list"
	"crypto/sha1"
)

// Predefined user permission codes.
var Permissions map[string]int = map[string]int{
	"READ":       1,
	"WRITE":      2,
	"READ|WRITE": 3,
	"MASTER":     7,
}

// User keeps information about single configured user. 
type User struct {
	Name       string
	Secret     string
	Permission int
}

/*
Authenticate matches given secret with current user credentials. If user
defined secret is empty, or given secret matches the defined one then returns
true, otherwise returns false.
*/
func (u *User) Authenticate(secret string) bool {
	return (u.Secret == "" || u.Secret == secret)
}

// IsAllowed checks if the user is permitted to do given operation.
func (u *User) IsAllowed(permission int) bool {
	return (u.Permission & permission == permission)
}

// userMap contains list of configured user entries.
type userMap map[string]*User

// Wrapper for standard websocket.Conn structure.
type conn struct {
	*websocket.Conn
	token   string
	session *User
}

// generateUniqueToken creates unique token using system `/dev/urandom`.
func generateUniqueToken() string {
	f, _ := os.OpenFile("/dev/urandom", os.O_RDONLY, 0) 
	b := make([]byte, 16) 
	f.Read(b) 
	f.Close() 
	token := sha1.New()
	token.Write(b)
	return fmt.Sprintf("%x", token.Sum())
}

/*
wrapConn wraps standard websocket connection object into one adjusted for
webrocker server funcionalities.
*/
func wrapConn(ws *websocket.Conn) *conn {
	return &conn{Conn: ws, token: generateUniqueToken()}
}

// sessionMap contains clients' sessions authenticated on this server.
type sessionMap map[*conn]*User

// connectionMap contains all client connections.
type connectionMap map[string]*conn

/*
Payload is an general strucutre for all sent event messages.

Simple examples how to create new event message:

    Payload SimpleMessage  = Payload("hello": "world")
    Payload ComplexMessage = Payload("hello": Data{"foo": "bar"})
*/
type Payload map[string]interface{}

// Data is an general structure for all received event messages.
type Data map[string]interface{}

// Returns name of event represented by this payload.
func (p *Payload) Event() (string, os.Error) {
	for k := range *p {
		return k, nil
	}
	return "", os.NewError("No event specified")
}

// Returns data contained by this payload.
func (p *Payload) Data() (*Data, os.Error) {
	for _, v := range *p {
		var d Data
		val, ok := v.(map[string]interface{})
		if ok {
			d = val
			return &d, nil
		}
		_, ok = v.(bool)
		if ok {
			return &d, nil
		}
	}
	return nil, os.NewError("Invalid format of the data")
}

// Predefined reply for successfull authentication.
func Authenticated(accessType string) *Payload {
	return &Payload{"authenticated": accessType}
}

// Predefined reply for successfull subscription.
func Subscribed(channelName string) *Payload {
	return &Payload{"subscribed": channelName}
}

// Predefined reply for successfull unsubscription.
func Unsubscribed(channelName string) *Payload {
	return &Payload{"unsubscribed": channelName}
}

// Predefined reply for successfull broadcast.
func Broadcasted(channelName string) *Payload {
	return &Payload{"broadcasted": channelName}
}

// Other predefined payloads.
var (
	LoggedOut Payload = Payload{"loggedOut": true}
)

// Error payloads.
var (
	InvalidPayload       Payload = Payload{"err": "INVALID_PAYLOAD"}
	InvalidUser          Payload = Payload{"err": "INVALID_USER"}
	InvalidCredentials   Payload = Payload{"err": "INVALID_CREDENTIALS"}
	InvalidChannel       Payload = Payload{"err": "INVALID_CHANNEL"}
	AccessDenied         Payload = Payload{"err": "ACCESS_DENIED"}
)

// Default handler, with various message codecs support.
type handler struct {
	Codec         websocket.Codec
	Users         userMap
	Log           *log.Logger
	server        *Server
	handler       websocket.Handler
	path          string
	registered    bool
	connections   connectionMap
	channels      channelMap
	subscriptions *list.List
}

/*
Creates new handler based on specified websocket's codec. Here's an trivial example:

     server := webrocket.NewServer("localhost:8080")
     handler := webrocket.NewHandler(websocket.JSON)
     server.Handle("/echo", handler)
     server.ListenAndServe()
*/
func NewHandler(codec websocket.Codec) *handler {
	return &handler{Codec: codec}
}

/*
Register initializes new handle under specified id (in this case an id is query path),
and returns valid websocket.Handler clojure to handle incoming messages.
*/
func (h *handler) Register(s *Server, id interface{}) (websocket.Handler, os.Error) {
	if h.registered {
		return nil, os.NewError("Handler already registered")
	}
	if h.Log == nil {
		h.Log = log.New(os.Stderr, id.(string)+" : ", log.LstdFlags)
	}
	h.server = s
	h.path = id.(string)
	h.handler = func(ws *websocket.Conn) {
		wrapped := wrapConn(ws)
		h.eventLoop(wrapped)
	}
	h.connections = make(connectionMap)
	h.channels = make(channelMap)
	h.subscriptions = list.New()
	h.registered = true
	s.Log.Printf("Registered handler: %s\n", h.path)
	return h.handler, nil
}

func (h *handler) eventLoop(ws *conn) {
	h.onOpen(ws)
	for {
		var recv Payload
		err := h.Codec.Receive(ws.Conn, &recv)
		if err != nil {
			if err == os.EOF {
				break
			}
			h.onError2(ws, InvalidPayload, err)
			continue
		}
		event, err := recv.Event()
		if err != nil {
			h.onError2(ws, InvalidPayload, err)
			continue
		}
		data, err := recv.Data()
		if err != nil {
			h.onError2(ws, InvalidPayload, err)
			continue
		}
		ok := h.dispatch(ws, event, data)
		if !ok {
			break
		}
	}
	h.onClose(ws)
}

// Dispatches receivied event to appropriate handler. 
func (h *handler) dispatch(ws *conn, event string, data *Data) bool {
	switch event {
	case "authenticate":
		h.onAuthenticate(ws, data)
	case "subscribe":
		h.onSubscribe(ws, data)
	case "unsubscribe":
		h.onUnsubscribe(ws, data)
	case "broadcast":
		h.onBroadcast(ws, data)
	case "logout":
		h.onLogout(ws)
	case "disconnect":
		h.onDisconnect(ws)
		return false
	}
	return true
}

// A helper for quick sending JSON-coded payloads to the connected client.
func (h *handler) send(ws *conn, data interface{}) os.Error {
	err := h.Codec.Send(ws.Conn, data)
	if err != nil {
		h.Log.Printf("[%s...] \033[35m~~> %s\033[0m\n", ws.token[:10], err.String())
	}
	return err
}

// A helper for checking if current connection is authorized to perform specified action.
func (h *handler) assertAccess(action string, ws *conn, permission int) bool {
	user := ws.session
	if user == nil || !user.IsAllowed(permission) {
		h.send(ws, AccessDenied)
		h.Log.Printf("[%s...] \033[33m[%s] Access denied\033[0m\n", ws.token[:10], action)
		return false
	}
	return true
}

/*
onError is a helper for dealing with failures caused mainly by invalid payload format,
or other message problems.
*/
func (h *handler) onError(ws *conn, payload Payload, errMsg string) os.Error {
	err := os.NewError(errMsg)
	h.onError2(ws, payload, err)
	return err
}

func (h *handler) onError2(ws *conn, payload Payload, err os.Error) {
	errName := payload["err"]
	h.send(ws, payload)
	h.Log.Printf("[%s...] \033[31m[%s] %s\033[0m\n", ws.token[:10], errName, err.String())
}

/*
onError3 is a helper for dealing with small errors, like failed authentication,
access denied errors etc.
*/
func (h *handler) onError3(ws *conn, payload Payload, errMsg string) os.Error {
	errName := payload["err"]
	err := os.NewError(errMsg)
	h.send(ws, payload)
	h.Log.Printf("[%s...] \033[33m[%s] %s\033[0m\n", ws.token[:10], errName, err.String())
	return err
}

/*
onOpen handles registration of the new connection in the system. Each new connection
has assigned SHA1 token to easily distinguish them from the others.
*/
func (h *handler) onOpen(ws *conn) {
	h.connections[ws.token] = ws
	h.Log.Printf("[%s...] \033[34m~~> Connected\033[0m\n", ws.token[:10])
}

func (h *handler) onClose(ws *conn) {
	h.Log.Printf("[%s...] \033[34m<~~ Disconnected\033[0m\n", ws.token[:10])
}

// onClose handles safe disconnection requested by the client.
func (h *handler) onDisconnect(ws *conn) {
	ws.Close()
}

// onAuthenticate manages session authentication for the connected client.
func (h *handler) onAuthenticate(ws *conn, data *Data) os.Error {
	userName, ok := (*data)["user"]
	if !ok {
		return h.onError(ws, InvalidPayload, fmt.Sprintf("Missing user name: %s", *data))
	}
	secret, ok := (*data)["secret"]
	if !ok {
		return h.onError(ws, InvalidPayload, fmt.Sprintf("Missing secret: %s", *data))
	}
	secretStr, ok := secret.(string)
	if !ok {
		return h.onError(ws, InvalidPayload, fmt.Sprintf("Invalid secret: %s", *data))
	}
	userNameStr, ok := userName.(string)
	if !ok {
		return h.onError(ws, InvalidPayload, fmt.Sprintf("Invalid user name: %s", *data))
	}
	user, ok := h.Users[userNameStr]
	if !ok {
		return h.onError(ws, InvalidUser, fmt.Sprintf("User does not exist: %s", *data))
	}
	ok = user.Authenticate(secretStr)
	if !ok {
		ws.session = nil
		return h.onError3(ws, InvalidCredentials, "Authentication failed")
	}
	ws.session = user
	h.send(ws, Authenticated(userNameStr))
	h.Log.Printf("[%s...] \033[36m~~> Authenticated as %s\033[0m\n", ws.token[:10], userNameStr)
	return nil
}

// onLogout finishes current session and unsubscribes all channels subscribed by the client. 
func (h *handler) onLogout(ws *conn) {
	ws.session = nil
	for e := h.subscriptions.Front(); e != nil; e = e.Next() {
		ch := e.Value.(*channel)
		ch.subscribe <- subscription{ws, false}
	}
	h.Log.Printf("[%s...] \033[34m<~~ Logged out\033[0m\n", ws.token[:10])
	h.send(ws, LoggedOut)
}

// onSubscribe handlers subscription of the specified channel.
func (h *handler) onSubscribe(ws *conn, data *Data) os.Error {
	ok := h.assertAccess("SUBSCRIBE", ws, Permissions["READ"])
	if !ok {
		return os.NewError("Access denied")
	}
	chanName, ok := (*data)["channel"]
	if !ok {
		return h.onError(ws, InvalidPayload, fmt.Sprintf("Missing channel name: %s", *data))
	}
	name, ok := chanName.(string)
	if !ok {
		return h.onError(ws, InvalidPayload, fmt.Sprintf("Invalid channel name: %s", *data))
	}
	if len(name) == 0 {
		return h.onError(ws, InvalidPayload, fmt.Sprintf("Invalid channel name: %s", *data))
	}
	ch, ok := h.channels[name]
	if !ok {
		ch = newChannel(h, name)
		h.channels[name] = ch
	}
	h.Log.Printf("[%s...] \033[32m[SUBSCRIBE ~> %s] Channel subscribed\033[0m\n", ws.token[:10], name)
	ch.subscribe <- subscription{ws, true}
	h.subscriptions.PushBack(ch)
	h.send(ws, Subscribed(name))
	return nil
}

// onUnsubscribe handles unsubscribing of the specified channel.
func (h *handler) onUnsubscribe(ws *conn, data *Data) os.Error {
	chanName, ok := (*data)["channel"]
	if !ok {
		return h.onError(ws, InvalidPayload, fmt.Sprintf("Missing channel name: %s", *data))
	}
	name, ok := chanName.(string)
	if !ok {
		return h.onError(ws, InvalidPayload, fmt.Sprintf("Invalid channel name: %s", *data))
	}
	if len(name) == 0 {
		return h.onError(ws, InvalidPayload, fmt.Sprintf("Invalid channel name: %s", *data))
	}
	ch, ok := h.channels[name]
	if ok {
		ch.subscribe <- subscription{ws, false}
		for e := h.subscriptions.Front(); e != nil; e = e.Next() {
			cmp := e.Value.(*channel)
			if ch == cmp {
				h.subscriptions.Remove(e)
			}
		}
		h.Log.Printf("[%s...] \033[32m[UNSUBSCRIBE ~> %s] Channel unsubscribed\033[0m\n", ws.token[:10], name)
	}
	h.send(ws, Unsubscribed(name))
	return nil
}

/*
onBroadcasts handles message from the current connection and spreads it out across all
clients subscribing specified channel.
*/
func (h *handler) onBroadcast(ws *conn, data *Data) os.Error {
	ok := h.assertAccess("BROADCAST", ws, Permissions["READ|WRITE"])
	if !ok { 
		return os.NewError("Access denied")
	}
	event, ok := (*data)["event"]
	if !ok {
		return h.onError(ws, InvalidPayload, fmt.Sprintf("Missing event name: %s", *data))
	}
	_, ok = event.(string)
	if !ok {
		return h.onError(ws, InvalidPayload, fmt.Sprintf("Invalid event name: %s", *data))
	}
	channel, ok := (*data)["channel"]
	if !ok {
		return h.onError(ws, InvalidPayload, fmt.Sprintf("Missing channel name: %s", *data))
	}
	channelStr, ok := channel.(string)
	if !ok {
		return h.onError(ws, InvalidPayload, fmt.Sprintf("Invalid channel name: %s", *data))
	}
	ch, ok := h.channels[channelStr]
	if !ok {
		return h.onError3(ws, InvalidChannel, fmt.Sprintf("Channel does not exist: %s", *data))
	}
	ch.broadcast <- func(reader *conn) {
		if reader != nil {
			h.send(reader, *data)
		}
	}
	h.Log.Printf("[%s...] \033[32m[BROADCAST => %s] Broadcasted: %s\033[0m\n", ws.token[:10], channelStr, *data)
	h.send(ws, Broadcasted(channelStr))
	return nil
}

// Creates new handler basd on the default JSON protocol.
func NewJSONHandler() *handler {
	return NewHandler(websocket.JSON)
}
