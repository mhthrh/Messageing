package main

import (
	"Github.com/mhthrh/EventDriven/Dispatcher/Controller"
	"Github.com/mhthrh/EventDriven/Model/Tool"
	"fmt"
	"log"
	"os"
	"os/signal"
)

var (
	sigint chan os.Signal
	wait   chan struct{}
	exit   chan struct{}
)

func init() {
	sigint = make(chan os.Signal)
	wait = make(chan struct{})
	exit = make(chan struct{})
}
func main() {
	tool, err := Tool.New(1)
	if err != nil {
		log.Fatalln(err)
	}

	c := Controller.New(*tool)
	go c.Run(&exit, &wait)

	go func() {
		signal.Notify(sigint, os.Interrupt)
		<-sigint
		exit <- struct{}{}
		fmt.Println("By,")

		//fmt.Println(string(debug.Stack()))

	}()
	<-wait
}
