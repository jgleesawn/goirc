package main

import (
	"fmt"
	//"net/http"
	"os"
	//"io/ioutil"
	"bufio"
	//"github.com/gorilla/websocket"
	//"github.com/jgleesawn/ECC_Conn"
	//"net"
	"math/rand"
	"strconv"
	"crypto/tls"
	//"time"
//	"io"
	//"encoding/json"
	"strings"
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

	blocking := make(chan bool)
	go func(blocking chan bool) {
		nrdr := bufio.NewReader(conn)
		for {
			blocking <- true
			packet, err := nrdr.ReadString('\n')
			if err != nil {
				fmt.Println(err)
				return
			}

			if packet[0:4] == "PING" {
				conn.Write([]byte("PONG"+packet[4:]))
			}

			nv := false
			nv = nv || strings.Contains(string(packet),"PING")
			nv = nv || strings.Contains(string(packet),"JOIN")
			nv = nv || strings.Contains(string(packet),"PART")
			nv = nv || strings.Contains(string(packet),"QUIT")
			if nv { continue }

			out := strings.Split(string(packet[:len(packet)-1]),":")
			fmt.Println(out[len(out)-1])
		}
	} (blocking)

	//prefix := ":irc.freenode.net/testing1234 "
	suffix := "\r\n"


	//var buf []byte
	rstdin := bufio.NewReader(os.Stdin)
	//blocking is an inefficient block so USER and NICK calls don't happen before ident information is received from server.
	<-blocking
	<-blocking
	<-blocking
	<-blocking
	conn.Write([]byte("USER a"+strconv.Itoa(rand.Int())+" * * :a"+strconv.Itoa(rand.Int())+suffix))
	conn.Write([]byte("NICK a"+strconv.Itoa(rand.Int())+suffix))

	go func(blocking chan bool) {for {<-blocking} } (blocking)


	for {
		rstr,_ := rstdin.ReadString('\n')
		str := rstr[0:len(rstr)-1] + suffix
		fmt.Println(str)
		_,err := conn.Write([]byte(str))
		if err != nil {
			fmt.Println(err)
		}

	}

//Look into WebSocket Keys, not sure using the same one every time is good.
}
