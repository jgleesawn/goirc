package main

import (
	"strings"
	"bufio"
	"fmt"
	"io"
	"net"
	"sort"
)

type ircpacket struct {
	prefix	string
	cmd		string
	params	string
	trail	string
}
func (ip *ircpacket) ToString() string {
	var strs []string
	if len(ip.prefix) > 0 {
		strs = append(strs,":"+ip.prefix)
	}
	strs = append(strs,ip.cmd)
	if len(ip.params) > 0 {
		strs = append(strs,ip.params)
	}
	if len(ip.trail) > 0 {
		strs = append(strs,":"+ip.trail)
	}
	str := strings.Join(strs," ")
	return str+"\r\n"
}

func NewIrcPacket(s string) ircpacket {
	var pkt ircpacket
	var split []string

	ti := strings.Index(s," :") + 2
	if ti < 2 {
		pkt.trail = ""
		ti = len(s)
	} else {
		pkt.trail = s[ti:]
		ti = ti-2
	}

	split = strings.Fields(s[:ti])

	off := 0
	if s[0] == ':' {
		off = 1
		pkt.prefix = strings.ToLower(split[0])
	} else {
		pkt.prefix = ""
	}
	pkt.cmd = strings.ToLower(split[off])
	pkt.params = strings.Join(split[off+1:]," ")
	return pkt
}

type Channel []string
/*
	text	[]string
	current	bool
	update	chan bool
}
*/

//Use SortedSlice instead of map for Channels, sort by channel name
type IrcClient struct {
	Conn		*net.Conn
	Channels	map[string]Channel
	current		string
	updateView	chan bool
	//Reader		bufio.Reader
	//Writer		bufio.Writer
}
func NewClient(conn net.Conn) IrcClient {
	var ic IrcClient
	ic.Conn = &conn
	ic.Channels = make(map[string]Channel)
	ic.updateView = make(chan bool,1)
	return ic
}
func (ic *IrcClient) Receive() {
	reader := bufio.NewReader(*ic.Conn)
	for {
		packet,err := reader.ReadString('\n')
		if err != nil {
			fmt.Println(err)
			return
		}
		pkt := NewIrcPacket(packet[:len(packet)-1])
		switch pkt.cmd {
		case "ping":
			var op ircpacket
			op.cmd = "pong"
			ic.Send(op)
			break
		case "join":
		case "part":
		case "quit":
			break
		case "privmsg":
			var msg string
			nickind := strings.Index(pkt.prefix,"!")
			if nickind != -1 {
				nick := pkt.prefix[1:nickind]
				msg = msg+nick
			}
			msg = msg+":"+pkt.trail
			ic.Channels[pkt.params] = append(ic.Channels[pkt.params],msg)
			if pkt.params == ic.current {
				ic.updateView <- true
			}
			break
		default:
			fmt.Println(pkt)
			break
		}
	}
}
func (ic *IrcClient) View() {
	prev := make(map[string]int)
	
	for {
		<-ic.updateView
		ch,ok := ic.Channels[ic.current]
		if ok {
			for _,l := range ch[prev[ic.current]:] {
				fmt.Println(l)
			}
			prev[ic.current] = len(ch)
		}
	}
}
func (ic *IrcClient) ProcessInput(inp io.Reader) {
	reader := bufio.NewReader(inp)
	for {
		var pkt ircpacket
		var ci int
		var pi int

		line,err := reader.ReadString('\n')
		if err != nil { continue }

		if len(line) == 0 { continue }

		pkt.prefix = ""
		pkt.cmd = ""
		if line[0] == '/' {
			ci = strings.Index(line," ")
			if ci == -1 { ci = len(line)-1 }
			pkt.cmd = line[1:ci]

			pi = strings.Index(line[ci+1:]," :")
			if pi != -1 {
				pkt.params = line[ci+1:ci+1+pi]
				pkt.trail = line[ci+1+pi+2:]
			} else {
				pkt.params = line[ci+1:]
			}
		}
		params := strings.Fields(pkt.params)

		//fmt.Println(ic.current)
		switch pkt.cmd {
		case "join":
			if len(params) == 0 { break }
			ic.Channels[params[0]] = append(ic.Channels[params[0]],params[0])
			ic.current = params[0]
			break
		case "part":
			if len(params) == 0 { break }
			_,ok := ic.Channels[params[0]]
			if ok {
				var chlist []string
				for k,_ := range ic.Channels {
					chlist = append(chlist,k)
				}
				sort.Sort(sort.StringSlice(chlist))
				for i,ch := range chlist {
					if ch == ic.current {
						if i > 0 {
							ic.current = chlist[i-1]
						} else {
							ic.current = chlist[0]
						}
						break
					}
				}
				delete(ic.Channels,params[0])
			}
			break
		case "msg":
			pkt.cmd = "privmsg"
		case "privmsg":
			pkt.cmd = "privmsg"
			break
		case "quit":
			return
		case "":
			pkt.cmd = "privmsg"
			pkt.params = ic.current
			pkt.trail = line
			break
		case "next":
			var chlist []string
			for k,_ := range ic.Channels {
				chlist = append(chlist,k)
			}
			sort.Sort(sort.StringSlice(chlist))
			for i,ch := range chlist {
				if ch == ic.current {
					if i < len(chlist)-1 {
						ic.current = chlist[i+1]
					} else {
						ic.current = chlist[0]
					}
					break
				}
			}
			fmt.Println(ic.current)
			ic.updateView <- true
			continue
		case "print":
			for _,l := range ic.Channels[ic.current] {
				fmt.Println(l)
			}
		default:
		}
		ic.Send(pkt)
	}
}

func (ic *IrcClient) Send(pkt ircpacket) {
	//fmt.Println(pkt.ToString())
	(*ic.Conn).Write([]byte(pkt.ToString()))
}





