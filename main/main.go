package main

import (
	"encoding/json"
	"fmt"
	"log"
	"os"

	"github.com/achilleasa/usrv"
	"github.com/achilleasa/usrv/middleware"
	"github.com/achilleasa/usrv/transport"
)

type ReqMsg struct {
	Key   string `json:"key"`
	Value string `json:"value"`
}

type ResMsg struct {
	Join string `json:"join"`
}

func foo(req *ReqMsg, res *ResMsg) error {
	res.Join = req.Key + req.Value
	return nil
}

func main() {
	logger := log.New(os.Stdout, "[APP] ", log.LstdFlags)

	//transp := transport.NewHttp(8080)
	transp := transport.NewInMemory()

	srv := usrv.NewServer("localhost:8080", transp)
	srv.Handle("com.foo",
		middleware.LogRequest(
			logger,
			middleware.JsonHandler(foo),
		),
	)
	srv.Listen()

	cli := usrv.NewClient("localhost:8080", transp)
	msg := cli.NewMessage("api", "com.foo")

	rmsg := ReqMsg{
		Key:   "hello ",
		Value: " world",
	}
	rmsgBytes, err := json.Marshal(rmsg)
	if err != nil {
		logger.Fatal(err)
	}

	msg.SetContent(rmsgBytes, nil)

	// Send
	replyChan := cli.Send(msg)
	resMsg := <-replyChan

	content, err := resMsg.Content()
	if err != nil {
		fmt.Printf("Got err: %v\n", err)
	} else {
		fmt.Printf("Got res: %#v\n", string(content))
	}
}
