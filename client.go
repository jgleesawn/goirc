package main

import (
	"fmt"
	//"net/http"
	//"net"
	"os"
	//"io/ioutil"
	//"bufio"
	//"github.com/gorilla/websocket"
	//"github.com/jgleesawn/ECC_Conn"
	//"net"
	"math/rand"
	"strconv"
	"crypto/tls"
	"time"
	//"time"
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
	}
	Client := NewClient(conn)

	go Client.Receive()
	go InitUserNick(Client)
	go Client.View()

	Client.ProcessInput(os.Stdin)
}
func InitUserNick(Client IrcClient) {
	time.Sleep(10)
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
