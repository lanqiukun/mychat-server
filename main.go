package main

import (
	"log"
	"math/rand"
	"net/http"
	"sync"
	"time"

	"github.com/gorilla/websocket"
)

var cp = make(map[uint64]*websocket.Conn)
var cpl sync.RWMutex

var cmp = make(chan ClientMessage, 1000000)
var cmpl sync.RWMutex

func detect() {
	i := 0
	for {
		i++
		time.Sleep(time.Second * 2)
		cpl.RLock()
		for i, _ := range cp {
			print(i)
			print("    ")
		}
		cpl.RUnlock()
		println()
	}
}

func init() {
	rand.Seed(time.Now().Unix())
	rand.Seed(int64(rand.Int31() + rand.Int31()))
}

func main() {

	go detect()

	go writePump()

	http.HandleFunc("/", serveHome)
	http.HandleFunc("/userinfo", userinfo)
	http.HandleFunc("/wstokenuserid", wstokenuserid)
	http.HandleFunc("/ws", establishwsconn)

	err := http.ListenAndServe("10.255.0.118:8080", nil)
	if err != nil {
		log.Fatal("ListenAndServe: ", err)
	}
}
