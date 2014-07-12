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

type ircpacket struct {
	prefix	string
	command	string
	params	string
	trail	string
}
func NewIrcPacket(s string) ircpacket {
	var pkt ircpacket
	ti := strings.Index(s," :") + 2
	if ti < 2 {
		ti = len(s)
		pkt.trail = ""
	} else {
		pkt.trail = s[ti:]
	}

	split := strings.Fields(s[:ti])

	off := 0
	if s[0] == ':' {
		off = 1
		pkt.prefix = split[0]
	} else {
		pkt.prefix = ""
	}
	pkt.command = split[off]
	pkt.params = strings.Join(split[off+1:]," ")
	return pkt
}

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
			pkt := NewIrcPacket(packet[:len(packet)-1])
			if pkt.command == "PING" {
				conn.Write([]byte("PONG"+packet[4:]))
			}

			nv := false
			nv = nv || pkt.command == "PING"
			nv = nv || pkt.command == "JOIN"
			nv = nv || pkt.command == "PART"
			nv = nv || pkt.command == "QUIT"
			if nv { continue }

			out := "" //strings.Split(string(packet[:len(packet)-1]),":")
			nickind := strings.Index(pkt.prefix,"!~")
			if nickind != -1 {
				nick := pkt.prefix[1:nickind]
				out = strings.Join([]string{out,nick},"")
			}
			out = strings.Join([]string{out,pkt.trail},":\t")

			fmt.Println(out)
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
