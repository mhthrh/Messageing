package main

import (
	"Github.com/mhthrh/EventDriven/Gateway/View"
	"Github.com/mhthrh/EventDriven/Model/Tool"
	"Github.com/mhthrh/EventDriven/Utilitys/Config"
	"context"
	"fmt"
	"github.com/gorilla/mux"
	"log"
	"net/http"
	"os"
	"os/signal"
	"time"
)

var (
	servers []http.Server
	sigint  chan os.Signal
	wait    chan struct{}
	ip      string
	port    int
)

func init() {
	sigint = make(chan os.Signal)
	wait = make(chan struct{})
}

func main() {
	ddd := Config.WriteConfig()
	fmt.Println(ddd)
	toolAsync, err := Tool.New(1)
	if err != nil {
		log.Fatalln(err)
	}
	for i, value := range toolAsync.Config.Listeners {
		ip = value.Ip
		port = value.Port
		if ip == "" && port <= 0 {
			fmt.Sprintf("check ip and/or port:%s:%d\n", ip, port)
			continue
		}
		tool, err := Tool.New(i)
		if err != nil {
			log.Fatalln(err)
		}
		smAsync := mux.NewRouter()
		if value.Typ == "Sync" {
			View.RunApiOnRouterSync(smAsync, *tool)
		} else {
			View.RunApiOnRouterAsync(smAsync, *tool)
		}
		serverAsync := http.Server{
			Addr:         fmt.Sprintf("%s:%d", ip, port),
			Handler:      smAsync,
			ErrorLog:     nil, //log.New(LogUtil.LogrusErrorWriter{}, "", 0),
			ReadTimeout:  10 * time.Second,
			WriteTimeout: 20 * time.Second,
			IdleTimeout:  180 * time.Second,
		}
		fmt.Printf("%s:%d Server added to the pool\n", ip, port)

		servers = append(servers, serverAsync)
		go func(i string, p int) {
			fmt.Printf("%s:%d Server Started\n", i, p)
			if err := serverAsync.ListenAndServe(); err != http.ErrServerClosed {
				log.Fatalf("HTTP server ListenAndServe: %v", err)
			}
		}(ip, port)

	}

	go func() {
		signal.Notify(sigint, os.Interrupt)
		<-sigint
		for _, server := range servers {
			if err := server.Shutdown(context.Background()); err != nil {
				log.Printf("HTTP server Shutdown: %v", err)
			}
		}
		close(wait)
	}()
	<-wait
}
