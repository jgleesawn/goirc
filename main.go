package main

import (
	"fmt"
	//"net/http"
	//"net"
	//"os"	//Used with ProcessReader(os.Stdin)
	//"io/ioutil"
	//"bufio"
	//"github.com/gorilla/websocket"
	//"github.com/jgleesawn/ECC_Conn"
	//"net"
	"math/rand"
	"strconv"
	"crypto/tls"
	"time"
//	"io"
	//"encoding/json"
	//"strings"
)

func main() {
	var port string
	var url string
	port = ":6667" //plaintext
	port = ":6697" //ssl/tls
	url = "irc.freenode.net"

	/*
	cfg := tls.Config{
		ClientAuth: tls.NoClientCert,
		ServerName: "irc.freenode.net",
	}
	cfg.BuildNameToCertificate()
	fmt.Println(cfg)
	*/

	conn, err := tls.Dial("tcp", url+port,nil)
	if err != nil {
		fmt.Println(err)
		return
	}
	Client := NewClient(conn)

	isFinished := make(chan bool,1)

	go Client.Receive()
	go InitUserNick(Client)
	go Client.View(isFinished)
	time.Sleep(2*time.Second)	//Allows termbox to init before ProcessTermbox checks.

	//Client.ProcessReader(os.Stdin)
	go Client.ProcessTermbox()

	<-isFinished
	Client.LogAll()
	conn.Close()

}
func InitUserNick(Client IrcClient) {
	rand.Seed(time.Now().Unix())
	time.Sleep(5)
	var pkt ircpacket
	pkt.cmd = "USER"
	pkt.params = "a"+strconv.Itoa(rand.Int())+" * *"
	pkt.trail = "a"+strconv.Itoa(rand.Int())
	Client.Send(pkt)
	pkt.cmd = "NICK"
	pkt.params = "a"+strconv.Itoa(rand.Int())
	pkt.trail = ""
	Client.Send(pkt)
}
