package Message

import (
	"Github.com/mhthrh/EventDriven/Utilitys/Json"
	"github.com/pborman/uuid"
	"time"
)

type (
	Request struct {
		ClientId uuid.UUID `json:"clientId" validate:"required"`
		ServerId uuid.UUID
		DateTime string
		Text     string   `json:"text" validate:"required"`
		Sender   Sender   `json:"sender" validate:"required"`
		Receiver Receiver `json:"receiver"  validate:"required"`
	}
	Response struct {
		ClientId         uuid.UUID `json:"clientId"`
		ServerId         uuid.UUID `json:"serverId"`
		ReceivedDateTime string    `json:"receivedDateTime"`
		SendDateTime     string    `json:"sendDateTime"`
		Text             string    `json:"text"`
		Sender           string    `json:"sender"`
		Receiver         string    `json:"receiver"`
		Ack              Acknowledge
	}
	Sender struct {
		UserName string `json:"userName" validate:"required"`
		Token    string `json:"token" validate:"required"`
	}
	Receiver struct {
		UserName string `json:"userName" validate:"required"`
	}

	Acknowledge struct {
		ClientId      uuid.UUID `json:"clientId"`
		ServerId      uuid.UUID `json:"serverId"`
		ReceiveTime   string    `json:"receiveTime"`
		Type          string    `json:"type"`
		StatusCode    int       `json:"statusCode"`
		StatusMessage string    `json:"statusMessage"`
	}
)

func CreateStruct(msg string) (*Request, error) {
	var m Request
	if err := Json.New(nil, nil).Json2Struct([]byte(msg), &m); err != nil {
		return nil, err
	}
	m.ServerId = uuid.NewUUID()
	m.DateTime = time.Now().String()
	return &m, nil
}
