package main
import (
	"strings"
	"strconv"
	"bufio"
	"fmt"
	"io"
	"net"
	"sort"
	"time"
	"os"

	"github.com/nsf/termbox-go"
)

type Channel []string

//Use SortedSlice instead of map for Channels, sort by channel name
type IrcClient struct {
	Conn		*net.Conn
	Channels	map[string]Channel
	Input		[]rune
	current		string
	updateView	chan bool
	map_lock	chan bool
	frame_offset	int			//Should be in the Channel struct.
	//Reader		bufio.Reader
	//Writer		bufio.Writer
}
func NewClient(conn net.Conn) IrcClient {
	var ic IrcClient
	ic.Conn = &conn
	ic.Channels = make(map[string]Channel)
	ic.updateView = make(chan bool,1)
	ic.map_lock = make(chan bool,1)
	ic.Channels["default"] = append(ic.Channels["default"],"Welcome to IRC")
	ic.current = "default"
	ic.map_lock <- true
	return ic
}
func (ic *IrcClient) Receive() {
	reader := bufio.NewReader(*ic.Conn)
	for {
		packet,err := reader.ReadString('\n')
		if err != nil {
			ic.updateView <- false
			time.Sleep(2*time.Second)
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
			<-ic.map_lock
			ic.Channels[pkt.params] = append(ic.Channels[pkt.params],msg)
			ic.map_lock<-true
			if pkt.params == ic.current { ic.frame_offset += 1 }
			break
		default:
			<-ic.map_lock
			ic.Channels["default"] = append(ic.Channels["default"],pkt.ToString())
			ic.map_lock<-true
			if ic.current == "default" { ic.frame_offset += 1}
			//fmt.Println(pkt)
			break
		}
		ic.updateView <- true
	}
}
func (ic *IrcClient) View(finish chan bool) {
	err := termbox.Init()
	if err != nil {
		return
	}
	defer func() {
		recover()
		termbox.Close()
		finish <- true
	} ()

	/*
	go func() {
		for {
			time.Sleep(100*time.Millisecond)
			ic.updateView <- true
		}
	}()*/

	//prev := make(map[string]int)
	width,height := termbox.Size()
	termbox.SetCursor(0,height)
	for <-ic.updateView {
		//fb := termbox.CellBuffer()
		<-ic.map_lock
		ch,ok := ic.Channels[ic.current]
		ic.map_lock<-true
		if ok {
			termbox.Clear(termbox.ColorWhite,termbox.ColorBlack)
			linecount := height-2

//Outputs Line Wrapped lines stored in a channel.
			bh := 1
			for l := len(ch)-1; linecount > bh && l >= 0; l -= 1 {
				if l >= ic.frame_offset { continue }	//Skips to offset
				var cnt int
				for _,_ = range ch[l] { cnt += 1 }
				length := cnt
				bh = length/(width-12)
				cnt = 0
				for _,c := range ch[l] {
					off := cnt/(width-12)
					termbox.SetCell(cnt%(width-12),linecount-bh+off,c,termbox.ColorWhite,termbox.ColorBlack)
					cnt += 1
				}
				linecount -= 1+bh
			}
		}

//Channel Name
		cnt := 0
		for _,c := range ic.current {
			termbox.SetCell(cnt,0,c,termbox.ColorRed,termbox.ColorBlack)
			cnt += 1
		}

//Input
		cnt = 0
		for _,c := range ic.Input {
			//termbox.SetCell(i,height-2,rune(height),termbox.ColorWhite,termbox.ColorBlack)
			termbox.SetCell(cnt,height-1,c,termbox.ColorWhite,termbox.ColorBlack)
			cnt += 1
		}

//Channel List
		<-ic.map_lock
		h := 0
		for k,v := range ic.Channels{
			if k[0] != '#' { continue }
			w := 0
			for _,r := range k {
				if w >= 8 { continue }
				termbox.SetCell(w+width-12,h,r,termbox.ColorWhite,termbox.ColorBlack)
				w += 1
			}
			num := strconv.Itoa(len(v))
			w = 0
			for _,r := range num {
				termbox.SetCell(width-len(num)+w,h,r,termbox.ColorWhite,termbox.ColorBlack)
				w += 1
			}
			h += 1
		}
		ic.map_lock <- true
		termbox.Flush()
	}
	//termbox.Close()
	//finish <- true
}

/*
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
*/
func (ic *IrcClient) ProcessReader(inp io.Reader) {
	reader := bufio.NewReader(inp)
	for {
		line,err := reader.ReadString('\n')
		if err != nil { continue }

		if len(line) == 0 { continue }
		ic.Input = []rune(line)
		ic.ProcessInput()
	}
}
func (ic *IrcClient) ProcessTermbox() {
	defer func() { ic.updateView <- false } ()

	for termbox.IsInit {
		e := termbox.PollEvent()
		if e.Type == termbox.EventKey {
			if int(e.Ch) != 0 {
				ic.Input = append(ic.Input,e.Ch)
			} else {
				switch e.Key {
				case termbox.KeyBackspace:
					if len(ic.Input) > 0 {
						ic.Input = ic.Input[:len(ic.Input)-1]
					}
				case termbox.KeyEnter:
					ic.ProcessInput()
				case termbox.KeySpace:
					ic.Input = append(ic.Input,' ')

				case termbox.KeyInsert:	//KeyArrowUp rune is picked up
					ic.frame_offset -= 1
					if ic.frame_offset < 0 { ic.frame_offset = 0 }
				case termbox.KeyDelete: //KeyArrowDown rune is picked up
					ic.frame_offset += 1
					<-ic.map_lock
					l := len(ic.Channels[ic.current])
					ic.map_lock <- true
					if ic.frame_offset > l { ic.frame_offset = l }
				case termbox.KeyPgup:
					_,height := termbox.Size()
					ic.frame_offset -= height
					if ic.frame_offset < 0 { ic.frame_offset = 0 }
				case termbox.KeyPgdn:
					_,height := termbox.Size()
					ic.frame_offset += height
					<-ic.map_lock
					l := len(ic.Channels[ic.current])
					ic.map_lock <- true
					if ic.frame_offset > l { ic.frame_offset = l }

				case termbox.KeyHome:
					ic.frame_offset = 0
				case termbox.KeyEnd:
					<-ic.map_lock
					ic.frame_offset = len(ic.Channels[ic.current])
					ic.map_lock <- true
				case termbox.KeyCtrlQ:
					return
				default:
				}
			}
		}
		ic.updateView <- true
	}
}

func (ic *IrcClient) ProcessInput() {
	var pkt ircpacket
	var pi int
	line := string(ic.Input)
	ic.Input = []rune{}
	if len(line) == 0 { return }

	pkt.prefix = ""
	pkt.cmd = ""
	if line[0] == '/' {
		sep := strings.Fields(line[1:])
		pkt.cmd = sep[0]

		line = strings.Join(sep[1:]," ")
		pi = strings.Index(line," :")
		if pi != -1 {
			pkt.params = line[:pi]
			pkt.trail = line[pi+2:]
		} else {
			pkt.params = line
		}
	}
	params := strings.Fields(pkt.params)

	//fmt.Println(ic.current)
	switch pkt.cmd {
	case "join":
		if len(params) == 0 { break }
		<-ic.map_lock
		ic.Channels[params[0]] = append(ic.Channels[params[0]],params[0])
		ic.map_lock<-true
		ic.current = params[0]
		break
	case "part":
		if len(params) == 0 { break }
		<-ic.map_lock
		_,ok := ic.Channels[params[0]]
		ic.map_lock<-true

		if ok {
			var chlist []string

			<-ic.map_lock
			for k,_ := range ic.Channels {
				chlist = append(chlist,k)
			}
			ic.map_lock<-true

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
			<-ic.map_lock
			delete(ic.Channels,params[0])
			ic.map_lock<-true
		}
		break
	case "msg":
		pkt.cmd = "privmsg"
		<-ic.map_lock
		ic.Channels[params[0]] = append(ic.Channels[params[0]],line)
		ic.map_lock<-true
		if params[0] == ic.current { ic.frame_offset += 1 }
	case "privmsg":
		pkt.cmd = "privmsg"
		<-ic.map_lock
		ic.Channels[params[0]] = append(ic.Channels[params[0]],line)
		ic.map_lock<-true
		if params[0] == ic.current { ic.frame_offset += 1 }
		break
	case "quit":
		return
	case "":
		<-ic.map_lock
		ic.Channels[ic.current] = append(ic.Channels[ic.current],line)
		ic.map_lock<-true
		ic.frame_offset += 1
		pkt.cmd = "privmsg"
		pkt.params = ic.current
		pkt.trail = line
		break
	case "next":
		var chlist []string
		<-ic.map_lock
		for k,_ := range ic.Channels {
			chlist = append(chlist,k)
		}
		ic.map_lock<-true
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
		<-ic.map_lock
		v,_ := ic.Channels[ic.current]
		ic.frame_offset = len(v)
		ic.map_lock <- true

		//fmt.Println(ic.current)
		//ic.updateView <- true
		return
	case "show":
		<-ic.map_lock
		v,ok := ic.Channels[params[0]]
		ic.map_lock <- true
		if ok {
			ic.current = params[0]
			ic.frame_offset = len(v)
		}
		return
	case "l":
		list := []string{"List of Channels."}
		<-ic.map_lock
		for k,_ := range ic.Channels {
			list = append(list,k+"\t\t"+strconv.Itoa(len(k)))
		}
		ic.Channels["list"] = list
		ic.map_lock <- true
		ic.current = "list"
		return
	case "sync":
		termbox.Sync()
		return
	case "h":
		help := []string{"List of client commands."}
		help = append(help,"/h             :Shows you this list")
		help = append(help,"/l             :Lists channels you're connected to")
		help = append(help,"/log           :Logs channel conversations")
		help = append(help,"/sync          :Resync's terimnal buffer")
		help = append(help,"/show chname   :Focuses window on chname if you're connected")
		help = append(help,"/msg name :msg :Sends msg to name")
		help = append(help,"Page Up        :Page Up")
		help = append(help,"Page Down      :Page Down")
		help = append(help,"Insert         :Scroll Up")
		help = append(help,"Delete         :Scroll Down")
		<-ic.map_lock
		ic.Channels["help"] = help
		ic.map_lock <- true
		ic.current = "help"
		return
	case "log":
		<-ic.map_lock
		for k,v := range ic.Channels {
			if k[0] != '#' { continue }
			file,err := os.Create(k+"_log_"+time.Now().String())
			if err != nil {
				file.Close()
				continue
			}
			for _,l := range v {
				file.Write([]byte(l+"\n"))
			}
			file.Close()
		}
		ic.map_lock <- true
		return
	/*case "print":
		for _,l := range ic.Channels[ic.current] {
			fmt.Println(l)
		}
		return
		*/
	default:
	}
	ic.Send(pkt)
}

func (ic *IrcClient) Send(pkt ircpacket) {
	//fmt.Println(pkt.ToString())
	(*ic.Conn).Write([]byte(pkt.ToString()))
}






