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
	"errors"
	"fmt"
	"github.com/gorilla/websocket"
	"github.com/pborman/uuid"
	"github.com/streadway/amqp"
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
		ID            uuid.UUID
		UserName      string
		Token         string
		SenderQueue   string
		MainQueueNAme string
		ReadWs        chan string
		WriteWs       chan string
		ReadRabbit    chan string
		WriteRabbit   chan struct{ queue, message string }
		Connection    *websocket.Conn
		Control       *Controller
	}

	Controller struct {
		Rabbit     *RabbitMQ.RabbitMq
		Database   *DbPool.Pool
		Redis      *Redis.Client
		Validation *Validation.Validation
		Cnfg       *Config.Config
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
		ID:            uuid.NewUUID(),
		MainQueueNAme: r.Host,
		UserName:      r.Header.Get("X-Websocket-user"),
		Token:         r.Header.Get("X-Websocket-token"),
		SenderQueue:   fmt.Sprintf("output-%s", r.Header.Get("X-Websocket-user")),
		ReadWs:        make(chan string),
		WriteWs:       make(chan string),
		ReadRabbit:    make(chan string),
		WriteRabbit:   make(chan struct{ queue, message string }),
		Connection:    nil,
		Control:       &c,
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
		conn.Close()
		return
	}

	handler.Connection = conn

	if err = c.Rabbit.DeclareQueue(handler.MainQueueNAme); err != nil {
		conn.WriteMessage(websocket.CloseMessage, []byte("server error"))
		conn.Close()
		return
	}

	if err = c.Rabbit.DeclareQueue(handler.SenderQueue); err != nil {
		conn.WriteMessage(websocket.CloseMessage, []byte("server error"))
		conn.Close()
		return
	}
	go serveNewSocket(&handler)
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

func serveNewSocket(agent *GatewayDispatcher) {
	ch1 := make(chan error)

	ch2, ch4, ch5, ch6, ch7 := make(chan struct{}), make(chan struct{}), make(chan struct{}), make(chan struct{}), make(chan struct{})
	defer func() {
		close(ch1)
		close(ch2)
		//close(ch3)
		close(ch4)
		close(ch5)
		close(ch6)
		close(ch7)
		agent = nil
	}()
	//Ping Websocket
	go func(cnn *websocket.Conn, msg *chan error, exit chan struct{}) {
		defer cnn.Close()
		for {
			select {
			case <-exit:
				fmt.Println("exit connection ping")
				return
			case <-time.Tick(time.Millisecond * 100):
				if err := cnn.WriteControl(websocket.PingMessage, []byte{}, time.Now().Add(time.Millisecond*200)); err != nil {
					*msg <- fmt.Errorf("canot access to webSoket")
				}
			}
		}

	}(agent.Connection, &ch1, ch2)
	//Read WebSocket
	go func(cnn *websocket.Conn, msg chan string, exit chan struct{}) {
		defer func() {
			_ = cnn.Close()
			fmt.Println("close Read WebSocket")
		}()
		cnn.SetReadLimit(maxMessageSize)

		for {
			select {
			case <-exit:
				fmt.Println("exit websocket reader")
				return
			default:
				_, message, err := cnn.ReadMessage()
				if err != nil {
					if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
						log.Printf("error: %v", err)
					}
					continue
				}
				message = bytes.TrimSpace(bytes.Replace(message, newline, space, -1))
				msg <- string(message)
			}
		}

	}(agent.Connection, agent.ReadWs, ch4)
	//Write Websocket
	go func(conn *websocket.Conn, data *chan string, exit chan struct{}) {
		ticker := time.NewTicker(pingPeriod)
		defer func() {
			ticker.Stop()
			conn.Close()
			fmt.Println("close Write Websocket")
		}()
		for {
			select {
			case <-exit:
				return
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
	}(agent.Connection, &agent.WriteWs, ch6)
	//Write Queue
	go func(tt chan struct{ queue, message string }, exit chan struct{}) {
		defer fmt.Println("close Write Queue")
		for {
			select {
			case t, ok := <-tt:
				if !ok {

				}
				//err := agent.Control.Rabbit.Publish(fmt.Sprintf("output-%s", strings.Split(message, "#")[0]), Json.New(nil, nil).Struct2Json(strings.Split(message, "#")[1:]))
				//	ffff := Json.New(nil, nil).Struct2Json(t.message)
				err := agent.Control.Rabbit.Publish(t.queue, t.message)
				fmt.Println(t.queue, "  ", t.message)
				if err != nil {

				}

			case <-exit:
				return
			}
		}

	}(agent.WriteRabbit, ch7)
	//Read Queue
	go func(queue string, body *chan string, exit chan struct{}) {
		defer func() {
		}()
		connectRabbitMQ, err := amqp.Dial(fmt.Sprintf("amqp://%s:%s@%s:%d/", agent.Control.Cnfg.Rabbit.UserName, agent.Control.Cnfg.Rabbit.Password, agent.Control.Cnfg.Rabbit.Host, agent.Control.Cnfg.Rabbit.Port))
		defer connectRabbitMQ.Close()
		if err != nil {

		}
		channelRabbitMQ, err := connectRabbitMQ.Channel()
		defer channelRabbitMQ.Close()
		if err != nil {

		}

		messages, err := channelRabbitMQ.Consume(
			queue, // queue name
			"",    // consumer
			true,  // auto-ack
			false, // exclusive
			false, // no local
			false, // no wait
			nil,   // arguments
		)
		if err != nil {
			return
		}
		for {
			select {
			case msg, ok := <-messages:
				if !ok {
					fmt.Println(err)
					continue
				}
				*body <- string(msg.Body)
				fmt.Println(queue, "")
			case <-exit:
				return
			}
		}

	}(agent.SenderQueue, &agent.ReadRabbit, ch5)

	for {
		select {
		//listener for receive exit signal
		case message, ok := <-ch1:
			if message != nil || !ok {
				ch2 <- struct{}{}
				ch4 <- struct{}{}
				ch5 <- struct{}{}
				ch6 <- struct{}{}
				ch7 <- struct{}{}
				fmt.Println(message)
				return
			}
		case message, ok := <-agent.ReadWs:
			if !ok {

			}
			m, err := Message.CreateStruct(message)
			if err != nil {
				agent.WriteWs <- err.Error()
				fmt.Println(err)
				continue
			}
			Ctx, _ := context.WithTimeout(context.Background(), time.Second*10)
			db := agent.Control.Database.Pull()
			User.New(db, agent.Control.Redis, &Ctx)
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
				agent.Control.Database.Push(db)
				agent.WriteWs <- Json.New(nil, nil).Struct2Json(&Message.Acknowledge{
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
			agent.Control.Database.Push(db)
			if !valid {
				agent.WriteWs <- "Token is not valid"
				ch1 <- errors.New("token is not valid")
				continue
			}
			if err != nil {
				agent.WriteWs <- Json.New(nil, nil).Struct2Json(&Message.Acknowledge{
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
			agent.WriteRabbit <- struct{ queue, message string }{queue: agent.MainQueueNAme, message: Json.New(nil, nil).Struct2Json(&m)}

			if err != nil {
				agent.WriteWs <- Json.New(nil, nil).Struct2Json(&Message.Acknowledge{
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
			agent.WriteWs <- Json.New(nil, nil).Struct2Json(&Message.Acknowledge{
				ClientId:      m.ClientId,
				ServerId:      m.ServerId,
				ReceiveTime:   m.DateTime,
				Type:          "Received By System",
				StatusCode:    0,
				StatusMessage: "Success",
			})
		case message, ok := <-agent.ReadRabbit:
			if !ok {

			}
			if !strings.HasPrefix(message, "102030") {
				agent.WriteWs <- message
				continue
			}
			var res Message.Response

			if err := Json.New(nil, nil).Json2Struct([]byte(strings.Replace(message, "102030", "", 1)), &res); err != nil {

			}
			agent.WriteRabbit <- struct{ queue, message string }{queue: fmt.Sprintf("output-%s", res.Sender), message: Json.New(nil, nil).Struct2Json(&res.Ack)}
			res.Ack = Message.Acknowledge{
				ClientId:      nil,
				ServerId:      nil,
				ReceiveTime:   "",
				Type:          "",
				StatusCode:    0,
				StatusMessage: "",
			}
			agent.WriteWs <- Json.New(nil, nil).Struct2Json(&res)

		}
	}

}
