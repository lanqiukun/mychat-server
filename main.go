package main

import (
	crypt_rand "crypto/rand"
	"fmt"
	"io"
	"log"
	"math/rand"
	"net/http"
	"sync"
	"time"

	"github.com/gorilla/websocket"

	"github.com/jinzhu/gorm"
	_ "github.com/jinzhu/gorm/dialects/mysql"
)

var connectionPool = make(map[uint64]*websocket.Conn)
var connectionPoolLock sync.RWMutex

var clientMessagePool = make(chan ClientMessage, 1000000)
var clientMessagePoolLock sync.RWMutex

var writeLock sync.RWMutex

func getdb() (*gorm.DB, error) {
	return gorm.Open("mysql", "root:Edison3306#@(116.85.40.216:3306)/mychat?charset=utf8mb4&parseTime=True&loc=Local")
}

func detect() {
	i := 0
	for {
		i++
		time.Sleep(time.Second * 2)
		connectionPoolLock.RLock()
		for i, _ := range connectionPool {
			print(i)
			print("    ")
		}
		connectionPoolLock.RUnlock()
		println()
	}
}

func init() {
	rand.Seed(time.Now().Unix())
	rand.Seed(int64(rand.Int31() + rand.Int31()))

	assertAvailablePRNG()
}

const (
	//1是开发环境，0是生产环境
	environment = 1

	schema = "http"

	frontEndPort = "80"
	backEndPort  = "8080"
	devhost      = "192.168.31.253"
	onlinehost   = "10.255.0.118"

	ActivateEmailAccountUrl = "/activate-email-account"
)

var servehost string
var EmailActivatingHostUrl string
var EmailRegisterSucceedUrl string

func init() {
	if environment == 1 {
		servehost = devhost
	} else {
		servehost = onlinehost
	}

	EmailActivatingHostUrl = schema + "://" + servehost + ":" + backEndPort + ActivateEmailAccountUrl
	EmailRegisterSucceedUrl = schema + "://" + servehost + ":" + frontEndPort
}

func assertAvailablePRNG() {
	// 判断系统支不支持生成密码安全的（cryptographically secure）伪随机数
	buf := make([]byte, 1)

	_, err := io.ReadFull(crypt_rand.Reader, buf)
	if err != nil {
		//go所运行的系统不支持上述要求
		panic(fmt.Sprintf("crypto/rand is unavailable: Read() failed with %#v", err))
	}
}

func main() {

	go detect()

	go writePump()

	//获取某个联系人的信息, 如果强调好友的信息时一般不用这个接口因为好友有备注名alias，而这个函数无法提供
	http.HandleFunc("/contactinfo", contactinfo)

	//验证用户凭据
	http.HandleFunc("/authenticationcredentials", authenticationcredentials)

	//邮箱账号注册
	http.HandleFunc("/email-register", emailRegister)

	//邮箱账户激活
	http.HandleFunc(ActivateEmailAccountUrl, activateEmailAccount)

	//验证邮箱和密码
	http.HandleFunc("/verifyemailpassword", verifyemailpassword)

	//获取所有联系人
	http.HandleFunc("/getcontact", getcontact)

	//获取当前用户的id和token
	http.HandleFunc("/getidtoken", getidtoken)

	//创建ws连接
	http.HandleFunc("/ws", establishWsConn)

	//文件上传
	http.HandleFunc("/upload", uploadFile)

	//文件访问
	http.Handle("/upload/", http.StripPrefix("/upload/", http.FileServer(http.Dir("./upload"))))

	err := http.ListenAndServe(servehost+":"+backEndPort, nil)
	if err != nil {
		log.Fatal("ListenAndServe: ", err)
	}
}
