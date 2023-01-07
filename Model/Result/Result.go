package Result

import (
	"Github.com/mhthrh/EventDriven/Utilitys/Json"
	"net/http"
	"time"
)

type Response struct {
	Header struct {
		Status int
	}
	Body struct {
		Status  int
		Time    time.Time
		Message interface{}
	}
}

func New(BStatus, HStatus int, result interface{}) *Response {
	r := new(Response)
	r.Body.Time = time.Now()
	r.Header.Status = HStatus
	r.Body.Status = BStatus
	r.Body.Message = result
	return r
}
func (r *Response) SendResponse(w http.ResponseWriter) {
	w.WriteHeader(r.Header.Status)
	w.Write([]byte(Json.New(nil, nil).Struct2Json(r.Body)))
	w.Header().Set("Content-Type", "application/json; charset=UTF-8")

}
