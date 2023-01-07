package Controller

import (
	"Github.com/mhthrh/EventDriven/Model/Message"
	"Github.com/mhthrh/EventDriven/Model/Request"
	"Github.com/mhthrh/EventDriven/Model/Result"
	"Github.com/mhthrh/EventDriven/Model/User"
	"Github.com/mhthrh/EventDriven/Utilitys/Config"
	"Github.com/mhthrh/EventDriven/Utilitys/DbPool"
	"Github.com/mhthrh/EventDriven/Utilitys/Json"
	"Github.com/mhthrh/EventDriven/Utilitys/RabbitMQ"
	"Github.com/mhthrh/EventDriven/Utilitys/Redis"
	"Github.com/mhthrh/EventDriven/Utilitys/Validation"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"github.com/gorilla/websocket"
	"github.com/pborman/uuid"
	"log"
	"net/http"
	"strings"
	"time"
)

const (
	writeWait      = 10 * time.Second
	pongWait       = 60 * time.Second
	pingPeriod     = (pongWait * 9) / 10
	maxMessageSize = 512
)

var (
	newline  = []byte{'\n'}
	space    = []byte{' '}
	upgrader = websocket.Upgrader{
		ReadBufferSize:  1024,
		WriteBufferSize: 1024,
	}
)

type (
	GenericError struct {
		Message string `json:"message"`
	}
	Key struct {
	}
	GatewayDispatcher struct {
		ID          uuid.UUID
		UserName    string
		Token       string
		ReadWs      chan string
		WriteWs     chan string
		ReadRabbit  chan string
		WriteRabbit chan string
		Connection  *websocket.Conn
	}

	Controller struct {
		Rabbit     *RabbitMQ.RabbitMq
		Database   *DbPool.Pool
		Redis      *Redis.Client
		Validation *Validation.Validation
		Cnfg       *Config.Config
	}

	Agent struct {
		cont *Controller
		conn *websocket.Conn
	}
)

func init() {

}
func New(rabbit *RabbitMQ.RabbitMq, database *DbPool.Pool, redis *Redis.Client, validation *Validation.Validation, cnfg *Config.Config) *Controller {
	return &Controller{
		Rabbit:     rabbit,
		Database:   database,
		Redis:      redis,
		Validation: validation,
		Cnfg:       cnfg,
	}
}

func (c *Controller) MiddleWare(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {

		var (
			object  interface{}
			id      string
			request *http.Request
		)

		switch strings.ToLower(r.URL.Path) {
		case "/signup", "/usersearch", "/signin":
			var user User.User

			if err := json.NewDecoder(r.Body).Decode(&user); err != nil {
				Result.New(1001, http.StatusBadRequest, GenericError{Message: err.Error()}.Message).SendResponse(w)
				return
			}
			id = user.Header.RequestId.String()
			request = r.WithContext(context.WithValue(r.Context(), Key{}, user))
			object = user
		case "/ws":

		default:
			Result.New(-900, http.StatusNotFound, GenericError{Message: fmt.Errorf("address not found").Error()}.Message).SendResponse(w)
			return

		}
		exist, err := c.Redis.KeyExist(id)
		if err != nil {
			Result.New(1002, http.StatusInternalServerError, GenericError{Message: err.Error()}.Message).SendResponse(w)
			return
		}
		if exist != 0 {
			Result.New(1003, http.StatusBadRequest, GenericError{Message: fmt.Errorf("doublicated request").Error()}.Message).SendResponse(w)
			return
		}
		_ = c.Redis.Set(id, Json.New(nil, nil).Struct2Json(object))

		next.ServeHTTP(w, request)

	})
}

func (c Controller) Message(w http.ResponseWriter, r *http.Request) {

	handler := GatewayDispatcher{
		ID:          uuid.NewUUID(),
		UserName:    r.Header.Get("X-Websocket-user"),
		Token:       r.Header.Get("X-Websocket-token"),
		ReadWs:      make(chan string),
		WriteWs:     make(chan string),
		ReadRabbit:  make(chan string),
		WriteRabbit: make(chan string),
	}
	Ctx1, cancel := context.WithTimeout(context.Background(), time.Second*10)
	defer cancel()
	db1 := c.Database.Pull()
	User.New(db1, c.Redis, &Ctx1)

	sender := User.User{
		Header:   Request.Request{},
		Name:     "",
		LastName: "",
		UserName: handler.UserName,
		PassWord: "",
		Email:    "",
		Avatar:   "",
		Token:    handler.Token,
	}
	if exist := sender.ExistUser(); exist == false {
		c.Database.Push(db1)
		w.Write([]byte("Authenticate error"))
		w.WriteHeader(http.StatusForbidden)
		fmt.Println("Authenticate error")
		return
	}
	if valid, _ := sender.TokenIsValid(); !valid {
		c.Database.Push(db1)
		w.WriteHeader(http.StatusForbidden)
		w.Write([]byte("TokenInvalid"))
		fmt.Println("TokenInvalid")
		return
	}
	c.Database.Push(db1)
	upgrader.CheckOrigin = func(r *http.Request) bool {
		return true
	}

	conn, err := upgrader.Upgrade(w, r, nil)

	if err != nil {
		conn.WriteMessage(websocket.CloseMessage, []byte("bad request"))
		return
	}

	handler.Connection = conn

	go readWs(conn, &handler.ReadWs)
	go writeWs(conn, &handler.WriteWs)

	if err = c.Rabbit.DeclareQueue(r.Host); err != nil {
		conn.WriteMessage(websocket.CloseMessage, []byte("server error"))
		return
	}
	senderQueue := fmt.Sprintf("Output-%s", handler.UserName)

	if err = c.Rabbit.DeclareQueue(senderQueue); err != nil {
		conn.WriteMessage(websocket.CloseMessage, []byte("server error"))
		return
	}

	go c.Rabbit.Consumer(senderQueue, &handler.ReadRabbit)

	go func(agent *Agent) {
		go func(rr *websocket.Conn) {
			c := time.Tick(time.Millisecond * 100)
			for range c {
				err := rr.WriteControl(websocket.PingMessage, []byte{}, time.Now().Add(writeWait))
				if err != nil {
					fmt.Println("be gaaaaa")
				}
				_, byt, err := rr.ReadMessage()
				if err != nil {
					fmt.Println("be gaaaa raftim")
				}
				fmt.Println(string(byt))
			}
		}(conn)

		for {
			select {
			case message, ok := <-handler.ReadWs:
				if !ok {
					agent.conn.WriteMessage(websocket.CloseMessage, []byte{})
					return
				}
				m, err := Message.CreateStruct(message)
				if err != nil {
					handler.WriteWs <- err.Error()
					fmt.Println(err)
					continue
				}
				Ctx, _ := context.WithTimeout(context.Background(), time.Second*10)
				db := c.Database.Pull()
				User.New(db, c.Redis, &Ctx)
				sender := User.User{
					Header:   Request.Request{},
					Name:     "",
					LastName: "",
					UserName: m.Sender.UserName,
					PassWord: "",
					Email:    "",
					Avatar:   "",
					Token:    m.Sender.Token,
				}
				receiver := User.User{
					Header:   Request.Request{},
					Name:     "",
					LastName: "",
					UserName: m.Receiver.UserName,
					PassWord: "",
					Email:    "",
					Avatar:   "",
					Token:    "",
				}

				if exist := receiver.ExistUser(); exist == false {
					c.Database.Push(db)
					handler.WriteWs <- Json.New(nil, nil).Struct2Json(&Message.Acknowledge{
						ClientId:      m.ClientId,
						ServerId:      m.ServerId,
						ReceiveTime:   m.DateTime,
						Type:          "ack",
						StatusCode:    -1,
						StatusMessage: "receiver not valid",
					})
					continue
				}
				valid, err := sender.TokenIsValid()
				c.Database.Push(db)
				if err != nil {
					handler.WriteWs <- Json.New(nil, nil).Struct2Json(&Message.Acknowledge{
						ClientId:      m.ClientId,
						ServerId:      m.ServerId,
						ReceiveTime:   m.DateTime,
						Type:          "ack",
						StatusCode:    -1,
						StatusMessage: err.Error(),
					})
					fmt.Println(err)
					continue
				}
				if !valid {
					handler.WriteWs <- "Token is not valid"
					conn.WriteMessage(websocket.CloseMessage, []byte("Token is not valid"))
					return
				}

				message = Json.New(nil, nil).Struct2Json(&m)
				fmt.Println(r.Host, "==>", message)
				err = agent.cont.Rabbit.Publish(r.Host, message)
				if err != nil {
					handler.WriteWs <- Json.New(nil, nil).Struct2Json(&Message.Acknowledge{
						ClientId:      m.ClientId,
						ServerId:      m.ServerId,
						ReceiveTime:   m.DateTime,
						Type:          "ack",
						StatusCode:    -1,
						StatusMessage: err.Error(),
					})
					fmt.Println(err)
					continue
				}

				handler.WriteWs <- Json.New(nil, nil).Struct2Json(&Message.Acknowledge{
					ClientId:      m.ClientId,
					ServerId:      m.ServerId,
					ReceiveTime:   m.DateTime,
					Type:          "Received By System",
					StatusCode:    0,
					StatusMessage: "Success",
				})

			case message, ok := <-handler.ReadRabbit:
				if !ok {
					agent.conn.WriteMessage(websocket.CloseMessage, []byte{})
					return
				}

				if !strings.HasPrefix(message, "102030") {
					handler.WriteWs <- message
					continue
				}
				var res Message.Response

				dddd := strings.Replace(message, "102030", "", 1)
				if err := Json.New(nil, nil).Json2Struct([]byte(dddd), &res); err != nil {

				}
				_ = agent.cont.Rabbit.Publish(fmt.Sprintf("Output-%s", res.Sender), Json.New(nil, nil).Struct2Json(&res.Ack))
				res.Ack = Message.Acknowledge{
					ClientId:      nil,
					ServerId:      nil,
					ReceiveTime:   "",
					Type:          "",
					StatusCode:    0,
					StatusMessage: "",
				}
				handler.WriteWs <- Json.New(nil, nil).Struct2Json(&res)

			}
		}

	}(&Agent{
		cont: &c,
		conn: conn,
	})

}

func (c *Controller) SignUp(writer http.ResponseWriter, request *http.Request) {
	defer func() {
		err := recover()
		if err != nil {
			Result.New(-99, http.StatusBadRequest, Json.New(nil, nil).Struct2Json(err)).SendResponse(writer)
		}

	}()
	ctx, _ := context.WithTimeout(context.Background(), 30*time.Second)
	defer ctx.Done()
	u := request.Context().Value(Key{}).(User.User)
	errs := c.Validation.Validate(u)
	if errs != nil {
		Result.New(-99, http.StatusBadRequest, Json.New(nil, nil).Struct2Json(errs)).SendResponse(writer)
		return
	}
	db := c.Database.Pull()
	User.New(db, c.Redis, &ctx)
	err := u.SignUp()
	c.Database.Push(db)
	if err != nil {
		Result.New(-99, http.StatusBadRequest, err.Error()).SendResponse(writer)
		return
	}
	Result.New(1, http.StatusOK, "success").SendResponse(writer)
}

func (c *Controller) SignIn(writer http.ResponseWriter, request *http.Request) {
	defer func() {
		err := recover()
		if err != nil {
			Result.New(-99, http.StatusBadRequest, Json.New(nil, nil).Struct2Json(err)).SendResponse(writer)
		}

	}()
	ctx, _ := context.WithTimeout(context.Background(), 30*time.Second)
	defer ctx.Done()
	u := request.Context().Value(Key{}).(User.User)
	errs := c.Validation.SomeValidate(u, "UserName", "PassWord")
	if len(errs) > 0 {
		Result.New(-99, http.StatusBadRequest, Json.New(nil, nil).Struct2Json(errs)).SendResponse(writer)
		return
	}
	errs = c.Validation.SomeValidate(u.Header, "RequestId")
	if len(errs) > 0 {
		Result.New(-99, http.StatusBadRequest, Json.New(nil, nil).Struct2Json(errs)).SendResponse(writer)
		return
	}
	db := c.Database.Pull()
	User.New(db, c.Redis, &ctx)
	usr, err := u.SignIn()
	c.Database.Push(db)
	if err != nil {
		Result.New(-99, http.StatusBadRequest, err.Error()).SendResponse(writer)
		return
	}
	Result.New(1, http.StatusOK, Json.New(nil, nil).Struct2Json(usr)).SendResponse(writer)
}

func (c *Controller) Search(writer http.ResponseWriter, request *http.Request) {
	defer func() {
		err := recover()
		if err != nil {
			Result.New(-99, http.StatusBadRequest, Json.New(nil, nil).Struct2Json(err)).SendResponse(writer)
		}

	}()
	ctx, _ := context.WithTimeout(context.Background(), 30*time.Second)
	defer ctx.Done()
	u := request.Context().Value(Key{}).(User.User)
	errs := c.Validation.SomeValidate(u, "UserName")
	if len(errs) > 0 {
		Result.New(-99, http.StatusBadRequest, Json.New(nil, nil).Struct2Json(errs)).SendResponse(writer)
		return
	}
	errs = c.Validation.SomeValidate(u.Header, "RequestId")
	if len(errs) > 0 {
		Result.New(-99, http.StatusBadRequest, Json.New(nil, nil).Struct2Json(errs)).SendResponse(writer)
		return
	}
	db := c.Database.Pull()
	User.New(db, c.Redis, &ctx)
	exist := u.ExistUser()
	c.Database.Push(db)
	if exist == false {
		Result.New(-99, http.StatusBadRequest, "user not found").SendResponse(writer)
		return
	}
	Result.New(1, http.StatusOK, "username is available").SendResponse(writer)
}

func readWs(conn *websocket.Conn, data *chan string) {
	defer func() {
		conn.Close()
	}()
	conn.SetReadLimit(maxMessageSize)
	conn.SetReadDeadline(time.Now().Add(pongWait))
	conn.SetPongHandler(func(string) error {
		conn.SetReadDeadline(time.Now().Add(pongWait))
		return nil
	})
	for {
		_, message, err := conn.ReadMessage()
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				log.Printf("error: %v", err)
			}
			break
		}
		message = bytes.TrimSpace(bytes.Replace(message, newline, space, -1))
		*data <- string(message)
	}
}

func writeWs(conn *websocket.Conn, data *chan string) {
	ticker := time.NewTicker(pingPeriod)
	defer func() {
		ticker.Stop()
		conn.Close()
	}()
	for {
		select {
		case message, ok := <-*data:
			conn.SetWriteDeadline(time.Now().Add(writeWait))
			if !ok {
				conn.WriteMessage(websocket.CloseMessage, []byte{})
				return
			}

			w, err := conn.NextWriter(websocket.TextMessage)
			if err != nil {
				return
			}
			w.Write([]byte(message))

			if err := w.Close(); err != nil {
				return
			}
		case <-ticker.C:
			conn.SetWriteDeadline(time.Now().Add(writeWait))
			if err := conn.WriteMessage(websocket.PingMessage, nil); err != nil {
				return
			}
		}
	}
}

func ping(ws *websocket.Conn, done chan struct{}) {
	ticker := time.NewTicker(pingPeriod)
	defer ticker.Stop()
	for {
		select {
		case <-ticker.C:
			if err := ws.WriteControl(websocket.PingMessage, []byte{}, time.Now().Add(writeWait)); err != nil {
				log.Println("ping:", err)
			}
		case <-done:
			return
		}
	}
}
