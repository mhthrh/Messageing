package Controller

import (
	"Github.com/mhthrh/EventDriven/Model/Message"
	"Github.com/mhthrh/EventDriven/Model/Tool"
	"Github.com/mhthrh/EventDriven/Utilitys/Config"
	"Github.com/mhthrh/EventDriven/Utilitys/DbPool"
	"Github.com/mhthrh/EventDriven/Utilitys/Json"
	"Github.com/mhthrh/EventDriven/Utilitys/RabbitMQ"
	"Github.com/mhthrh/EventDriven/Utilitys/Redis"
	"Github.com/mhthrh/EventDriven/Utilitys/Validation"
	"context"
	"fmt"
	"github.com/pborman/uuid"
	"sync"
	"time"
)

var (
	dic        map[string]chan string
	queues     chan []string
	goroutines map[int]chan struct{}
)

type Controller struct {
	Redis      *Redis.Client
	Validation *Validation.Validation
	Database   *DbPool.Pool
	Rabbit     *RabbitMQ.RabbitMq
	Cnfg       *Config.Config
}

func init() {
	dic = make(map[string]chan string)
	queues = make(chan []string)
	goroutines = map[int]chan struct{}{
		0: make(chan struct{}),
		1: make(chan struct{}),
	}
}
func New(tool Tool.Tool) *Controller {
	return &Controller{
		Redis:      tool.Redis,
		Validation: tool.Validation,
		Database:   tool.DB,
		Rabbit:     tool.Rabbit,
		Cnfg:       tool.Config,
	}
}

func (c Controller) Run(msg, wait *chan struct{}) {
	t1 := make(chan struct{})
	t2 := make(chan struct{})
	go c.Rabbit.Queues(&queues)

	go func(e, w chan struct{}) {
		for {
			select {
			case q := <-queues:
				for _, queue := range q {
					_, ok := dic[queue]
					if !ok {
						body := make(chan string)
						dic[queue] = body
						go c.Rabbit.Consumer(queue, &body)
					}
				}
			case <-e:
				w <- struct{}{}
				return
			}
		}
	}(goroutines[0], t1)

	go func(e, w chan struct{}) {
		var i int
		wait := sync.WaitGroup{}
		for {
			select {
			case <-e:
				wait.Wait()
				w <- struct{}{}
				return
			default:
				for _, value := range dic {
					select {
					case msg := <-value:
						fmt.Println(msg)
						i++
						wait.Add(i)
						go c.runMethod(msg, wait)
					case <-time.After(time.Microsecond):
					}

				}
			}
		}
	}(goroutines[1], t2)

	<-*msg

	for _, goroutine := range goroutines {
		goroutine <- struct{}{}
	}
	<-t1
	<-t2
	*wait <- struct{}{}
	//debug.PrintStack()
	//fmt.Println(string(debug.Stack()))
}
func (c Controller) runMethod(message string, wait sync.WaitGroup) {
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*10)
	defer cancel()
	defer wait.Done()
	var m Message.Request
	if err := Json.New(nil, nil).Json2Struct([]byte(message), &m); err != nil {
	}

	outReceiver := fmt.Sprintf("Output-%s", m.Receiver.UserName)
	outSender := fmt.Sprintf("Output-%s", m.Sender.UserName)

	err := c.Rabbit.DeclareQueue(outReceiver)
	if err != nil {

	}
	db := c.Database.Pull()
	_, err = db.Db.ExecContext(ctx, fmt.Sprintf("INSERT INTO `MyDb`.`messages`(`id`,`ins_date`,`msg_date`,`from`,`to`,`text`)VALUES('%s','%s','%s','%s','%s','%s')", uuid.NewUUID(), time.Now().String(), m.DateTime, m.Sender.UserName, m.Receiver.UserName, m.Text))
	c.Database.Push(db)
	if err != nil {

	}
	var r Message.Request
	err = Json.New(nil, nil).Json2Struct([]byte(message), &r)
	if err != nil {

	}
	err = c.Rabbit.Publish(outReceiver, "102030"+Json.New(nil, nil).Struct2Json(&Message.Response{
		ClientId:         r.ClientId,
		ServerId:         r.ServerId,
		ReceivedDateTime: r.DateTime,
		SendDateTime:     time.Now().String(),
		Text:             r.Text,
		Sender:           r.Sender.UserName,
		Receiver:         r.Receiver.UserName,
		Ack: Message.Acknowledge{
			ClientId:      r.ClientId,
			ServerId:      r.ServerId,
			ReceiveTime:   r.DateTime,
			Type:          "Received By Receiver",
			StatusCode:    0,
			StatusMessage: "",
		},
	}))
	if err != nil {

	}
	err = c.Rabbit.Publish(outSender, Json.New(nil, nil).Struct2Json(&Message.Acknowledge{
		ClientId:      m.ClientId,
		ServerId:      m.ServerId,
		ReceiveTime:   m.DateTime,
		Type:          "Send2Receiver",
		StatusCode:    0,
		StatusMessage: "Success",
	}))
	if err != nil {
	}

}
