package View

import (
	"Github.com/mhthrh/EventDriven/Gateway/Controller"
	"Github.com/mhthrh/EventDriven/Model/Result"
	"Github.com/mhthrh/EventDriven/Model/Tool"
	"github.com/gorilla/mux"
	http "net/http"
)

var (
	getServices, postServices *mux.Router
	controller                *Controller.Controller
)

func RunApiOnRouter(sm *mux.Router, tool Tool.Tool) {
	controller = Controller.New(tool.Rabbit, tool.DB, tool.Redis, tool.Validation, tool.Config)
	getServices = sm.Methods(http.MethodGet).Subrouter()
	postServices = sm.Methods(http.MethodPost).Subrouter()
	//sm.Use(controller.MiddleWare)

	getServices.HandleFunc("/message", controller.Message)

	postServices.HandleFunc("/signup", controller.SignUp)
	postServices.HandleFunc("/signin", controller.SignIn)
	postServices.HandleFunc("/usersearch", controller.Search)

	sm.NotFoundHandler = http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		Result.New(1003, http.StatusBadRequest, "bia boro to konam, kir shodi :)").SendResponse(writer)
		return
	})
}
