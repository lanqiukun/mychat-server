package main

import (
	"log"
	"math/rand"
	"net/http"
	"sync"
	"time"

	"github.com/gorilla/websocket"

	"github.com/jinzhu/gorm"
	_ "github.com/jinzhu/gorm/dialects/mysql"
)

var cp = make(map[uint64]*websocket.Conn)
var cpl sync.RWMutex

var cmp = make(chan ClientMessage, 1000000)
var cmpl sync.RWMutex

func getdb() (*gorm.DB, error) {
	return gorm.Open("mysql", "root:Edison3306#@(lowb.top:3306)/mychat?charset=utf8&parseTime=True&loc=Local")
}

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
	http.HandleFunc("/getcontact", getcontact)
	http.HandleFunc("/wstokenuserid", wstokenuserid)
	http.HandleFunc("/ws", establishwsconn)

	// err := http.ListenAndServe("10.255.0.118:8080", nil)

	err := http.ListenAndServe("192.168.31.253:8080", nil)
	if err != nil {
		log.Fatal("ListenAndServe: ", err)
	}
}
