package Request

import "github.com/pborman/uuid"

type (
	Request struct {
		ClientToken string    `json:"clientToken"`
		ClientIP    string    `json:"clientIP"`
		RequestId   uuid.UUID `json:"requestId" validate:"required"`
		serverId    uuid.UUID
		ClientTime  string `json:"clientTime" validate:"required"`
		serverTime  string
	}
)
