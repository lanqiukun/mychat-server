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

var writeLock sync.RWMutex

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

const (
	environment = 0
	devhost     = "localhost"
	onlinehost  = "10.255.0.118"
)

func main() {

	go detect()

	go writePump()

	//获取某个联系人的信息, 如果强调好友的信息时一般不用这个接口因为好友有备注名alias，而这个函数无法提供
	http.HandleFunc("/contactinfo", contactinfo)

	//验证用户凭据
	http.HandleFunc("/authenticationcredentials", authenticationcredentials)

	//获取所有联系人
	http.HandleFunc("/getcontact", getcontact)

	//获取当前用户的id和token
	http.HandleFunc("/getidtoken", getidtoken)

	//创建ws连接
	http.HandleFunc("/ws", establishWsConn)

	var servehost string

	if environment == 1 {
		servehost = devhost
	} else {
		servehost = onlinehost
	}

	err := http.ListenAndServe(servehost+":8080", nil)
	if err != nil {
		log.Fatal("ListenAndServe: ", err)
	}
}
