package main

import (
	"go-web-app/trace"
	"log"
	"net/http"

	"github.com/gorilla/websocket"
)

type room struct {
	forward chan []byte
	join    chan *client
	leave   chan *client
	clients map[*client]bool
	tracer  trace.Tracer
}

func newRoom() *room {
	return &room{
		forward: make(chan []byte),
		join:    make(chan *client),
		leave:   make(chan *client),
		clients: make(map[*client]bool),
	}
}

func (r *room) run() {
	for {
		select {
		case client := <-r.join:
			r.clients[client] = true
			r.tracer.Trace("Welcome new comer!")

		case client := <-r.leave:
			delete(r.clients, client)
			close(client.send)
			r.tracer.Trace("GoodBye client.")

		case msg := <-r.forward:
			for client := range r.clients {
				select {
				case client.send <- msg:
					// send msg
					r.tracer.Trace("The message was sent.")
				default:
					// fail to send msg
					delete(r.clients, client)
					close(client.send)
					r.tracer.Trace("Fail to send the message.")
				}
			}
		}
	}
}

const (
	socketBufferSize  = 1024
	messageBufferSize = 256
)

// to use WebSocket
var upgrader = &websocket.Upgrader{
	ReadBufferSize:  socketBufferSize,
	WriteBufferSize: socketBufferSize,
}

func (r *room) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	socket, err := upgrader.Upgrade(w, req, nil)
	if err != nil {
		log.Fatal("ServeHTTP:", err)
		return
	}

	client := &client{
		socket: socket,
		send:   make(chan []byte, messageBufferSize),
		room:   r,
	}

	r.join <- client // participate in chat room
	defer func() { r.leave <- client }()

	go client.write()
	client.read()
}
