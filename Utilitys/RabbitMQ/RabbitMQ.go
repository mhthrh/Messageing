package RabbitMQ

import (
	"Github.com/mhthrh/EventDriven/Utilitys/Config"
	"encoding/json"
	"fmt"
	"github.com/streadway/amqp"
	"net/http"
	"strings"
	"time"
)

type RabbitMq struct {
	Connection *amqp.Connection
	Channel    *amqp.Channel
	Cnfg       *Config.Config
}

func New(url string, cnfg *Config.Config) (*RabbitMq, error) {
	connectRabbitMQ, err := amqp.Dial(url)
	if err != nil {
		return nil, err
	}

	channelRabbitMQ, err := connectRabbitMQ.Channel()
	if err != nil {
		return nil, err
	}

	return &RabbitMq{
		Connection: connectRabbitMQ,
		Channel:    channelRabbitMQ,
		Cnfg:       cnfg,
	}, nil
}

func (mq *RabbitMq) DeclareQueue(queue string) error {
	_, err := mq.Channel.QueueDeclare(
		queue, // queue name
		true,  // durable
		false, // auto delete
		false, // exclusive
		false, // no wait
		nil,   // arguments
	)
	return err
}

func (mq *RabbitMq) Publish(queue string, msg string) error {

	err := mq.Channel.Publish(
		"",    // exchange
		queue, // queue name
		false, // mandatory
		false, // immediate
		amqp.Publishing{
			ContentType: "text/plain",
			Body:        []byte(msg),
		}, // message to publish
	)
	if err != nil {
		return fmt.Errorf("publish error2")
	}
	return nil

}

func (mq *RabbitMq) Consumer(queue string, body *chan string, exit chan struct{}) {
	defer fmt.Println("kire khar consume rabbit")
	messages, err := mq.Channel.Consume(
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

}

func (mq *RabbitMq) Queues(queues *chan []string) error {
	url := fmt.Sprintf("http://%s:%d/api/queues/", mq.Cnfg.Rabbit.Host, 15672)

	for range time.Tick(time.Millisecond * 2000) {
		q, err := queueList(url, mq.Cnfg.Rabbit.UserName, mq.Cnfg.Rabbit.Password)
		if err != nil {
			return err
		}
		*queues <- q
	}
	return nil
}

func queueList(url, user, pass string) ([]string, error) {
	var result []string
	var e error
	defer func() {
		err := recover()
		if err != nil {
			e = err.(error)
			result = nil
		}
	}()

	type Queue struct {
		Name  string `json:name`
		VHost string `json:vhost`
	}

	client := &http.Client{}
	req, _ := http.NewRequest("GET", url, nil)
	req.SetBasicAuth(user, pass)
	resp, _ := client.Do(req)

	value := make([]Queue, 0)
	json.NewDecoder(resp.Body).Decode(&value)
	for _, queue := range value {
		if !strings.HasPrefix(queue.Name, "output") {
			if !strings.HasPrefix(queue.Name, "amq.gen") {
				result = append(result, queue.Name)
			}
		}
	}
	return result, e
}
