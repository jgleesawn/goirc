package main
import (
	"strings"
	"strconv"
	"bufio"
	//"fmt"
	"io"
	"net"
	"sort"
	"time"
	"os"

	"github.com/nsf/termbox-go"
)

type Channel struct { //[]string
	Msgs			[]string
	Users			[]string
	Frame_offset	int
	important	bool
}

//Use SortedSlice instead of map for Channels, sort by channel name
type IrcClient struct {
	Name		string
	Conn		*net.Conn
	Channels	map[string]Channel
	Input		[]rune

	updateView	chan bool

	current		string
	map_lock	chan bool
	keywords	[]string
	running		bool
	//Reader		bufio.Reader
	//Writer		bufio.Writer
}
func (ic *IrcClient) AddUser(chname string,username string) {
	<-ic.map_lock

	ch,_ := ic.Channels[chname]
	ch.Users = append(ch.Users,username)
	sort.Sort(sort.StringSlice(ch.Users))
	ind := sort.StringSlice(ch.Users).Search(username)
	if ch.Users[ind] != username {
		l := append(ch.Users[:ind],username)
		if ind < len(ch.Users) {
			ch.Users = append(l,ch.Users[ind:]...)
		}
	}
	ic.Channels[chname] = ch

	ic.map_lock <- true
}
func (ic *IrcClient) RemUser(chname string,username string) {
	<-ic.map_lock
	
	ch,_ := ic.Channels[chname]
	if len(ch.Users) == 0 {
		ic.map_lock <- true
		return
	}
	ind := sort.StringSlice(ch.Users).Search(username)
	if ch.Users[ind] == username {
		if ind < len(ch.Users)-1 {
			ch.Users = append(ch.Users[:ind],ch.Users[ind+1:]...)
		} else {
			ch.Users = ch.Users[:ind]
		}
	}
	ic.Channels[chname] = ch

	ic.map_lock <- true
}
func (ic *IrcClient) AddMsg(chname string,msg string) {
	<-ic.map_lock

	ch,_ := ic.Channels[chname]
	ch.Msgs = append(ch.Msgs,msg)
	ch.Frame_offset += 1

	for _,w := range ic.keywords {
		if strings.Contains(msg,w) {
			ch.important = true
		}
	}

	ic.Channels[chname] = ch

	ic.map_lock <- true
}
func (ic *IrcClient) Scroll(dist int) {
	<-ic.map_lock
		ch,_ := ic.Channels[ic.current]
		ch.Frame_offset += dist
		l := len(ch.Msgs)
		if ch.Frame_offset > l {ch.Frame_offset = l}
		if ch.Frame_offset < 0 {ch.Frame_offset = 0}
		ic.Channels[ic.current] = ch
	ic.map_lock <- true
}
func (ic *IrcClient) NextChan() {
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
			<-ic.map_lock
			v,ok := ic.Channels[ic.current]
			if ok {
				v.important = false
				ic.Channels[ic.current] = v
			}
			ic.map_lock <- true
			break
		}
	}
}
func (ic *IrcClient) LogAll() {
	var tnames []string
	<-ic.map_lock
	for k,_ := range ic.Channels { tnames = append(tnames,k) }
	ic.map_lock <- true
	for _,n := range tnames { ic.Log(n) }
}
func (ic *IrcClient) Log(fn string) {
	<-ic.map_lock
	v,ok := ic.Channels[fn]
	ic.map_lock <- true
	if ok {
		if fn != "list" || fn != "help" {
		//if fn[0] == '#' {
			file,err := os.Create(fn+"_log_"+time.Now().String())
			if err != nil {
				file.Close()
				return
			}
			for _,l := range v.Msgs {
				file.Write([]byte(l+"\n"))
			}
			file.Close()
		}
	}
}

func NewClient(conn net.Conn) IrcClient {
	var ic IrcClient
	ic.Conn = &conn
	ic.Channels = make(map[string]Channel)
	ic.map_lock = make(chan bool,1)
	ic.updateView = make(chan bool,1)

	ic.map_lock <- true	//Required for AddMsg
	ic.AddMsg("default","Welcome to IRC")

	ic.current = "default"
	ic.running = true
	return ic
}
func (ic *IrcClient) Receive() {
	reader := bufio.NewReader(*ic.Conn)
	for ic.running {
		packet,err := reader.ReadString('\n')
		if err != nil {
			ic.running = false
			//ic.updateView <- false
			//fmt.Println(err)
			return
		}
		pkt := NewIrcPacket(packet[:len(packet)-1])
		switch pkt.cmd {
		case "ping":
			var op ircpacket
			op.cmd = "pong"
			ic.Send(op)
			break
		case "353":
			usernames := strings.Fields(pkt.trail[1:])
			params := strings.Fields(pkt.params)
			for _,u := range usernames {
				ic.AddUser(params[len(params)-1],u)
			}

		case "join":
			f := func(r rune) bool {return (r == '!' || r == '@')}
			name := strings.FieldsFunc(pkt.prefix,f)[0]
			params := strings.Fields(pkt.params)
			ic.AddUser(params[0],name)
		case "part":
			f := func(r rune) bool {return (r == '!' || r == '@')}
			name := strings.FieldsFunc(pkt.prefix,f)[0]
			params := strings.Fields(pkt.params)
			ic.RemUser(params[0],name)
		case "quit":
			break
		case "privmsg":
			var msg string
			f := func(r rune) bool {return (r == '!' || r == '@')}
			name := strings.FieldsFunc(pkt.prefix,f)[0]
			/*
			nickind := strings.Index(pkt.prefix,"!")
			if nickind != -1 {
				nick := pkt.prefix[1:nickind]
				msg = msg+nick
			}
			*/
			p := strings.Fields(pkt.params)
			if len(p) == 0 {
				ic.AddMsg("default",pkt.ToString())
				break
			}
			msg = msg+name
			msg = msg+":"+pkt.trail
			if p[len(p)-1] == ic.Name {
				ic.AddMsg(name,msg)
			} else {
				ic.AddMsg(pkt.params,msg)
			}
			break
		default:
			ic.AddMsg("default",pkt.ToString())
			//fmt.Println(pkt)
			break
		}
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
		ic.running = false
		finish <- true	//Used for blocking in main.
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
	fps := time.NewTicker(time.Second)
	for ic.running {
		select {
		case <-ic.updateView:
		case <-fps.C:
		}
		//fb := termbox.CellBuffer()
		<-ic.map_lock
		ch,ok := ic.Channels[ic.current]
		ic.map_lock<-true
		if ok {	//Should be fine even if you don't check. 
			termbox.Clear(termbox.ColorWhite,termbox.ColorBlack)
			linecount := height-2

//Outputs Line Wrapped lines stored in a channel.
			bh := 1
			for l := len(ch.Msgs)-1; linecount > bh && l >= 0; l -= 1 {
				if l >= ch.Frame_offset { continue }	//Skips to offset
				var cnt int
				for _,_ = range ch.Msgs[l] { cnt += 1 }
				length := cnt
				bh = length/(width-12)
				cnt = 0

				fg := termbox.ColorWhite
				for _,w := range ic.keywords {
					if strings.Contains(ch.Msgs[l],w) {
						fg = termbox.ColorGreen
					}
				}

				for _,c := range ch.Msgs[l] {
					off := cnt/(width-12)
					termbox.SetCell(cnt%(width-12),linecount-bh+off,c,fg,termbox.ColorBlack)
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
			if k == "default" { continue }
			//if k[0] != '#' { continue }
			w := 0
			fg := termbox.ColorWhite
			if v.important { fg = termbox.ColorRed }
			for _,r := range k {
				if w >= 8 { continue }
				termbox.SetCell(w+width-12,h,r,fg,termbox.ColorBlack)
				w += 1
			}
			num := strconv.Itoa(len(v.Msgs))
			w = 0
			for _,r := range num {
				termbox.SetCell(width-len(num)+w,h,r,fg,termbox.ColorBlack)
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
	for ic.running {
		line,err := reader.ReadString('\n')
		if err != nil { continue }

		if len(line) == 0 { continue }
		ic.Input = []rune(line)
		ic.ProcessInput()
	}
}
func (ic *IrcClient) ProcessTermbox() {
	defer func() { 
		ic.running = false
	} ()

	for e := termbox.PollEvent(); termbox.IsInit; e = termbox.PollEvent() {
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
				case termbox.KeyTab:	//Poor Tab-completion Doesn't cycle.
					f := strings.Fields(string(ic.Input))
					if len(f) == 0 { break }
					w := f[len(f)-1]

					<-ic.map_lock
						ch,_ := ic.Channels[ic.current]
					ic.map_lock <- true
					ind := sort.StringSlice(ch.Users).Search(w)
					if ind == len(ch.Users) { break }
					if strings.Contains(ch.Users[ind],w) {
						f[len(f)-1] = ch.Users[ind]
						ic.Input = []rune(strings.Join(f," "))
					}


				case termbox.KeyCtrlN:
					ic.NextChan()
				case termbox.KeyCtrlW:	//Not a good idea.
//Double buffering ic.Input could have out-of sync write/reads that throw it all to shit.
					backup := ic.Input
					ic.Input = []rune("/part "+ic.current)
					ic.ProcessInput()
					ic.Input = backup
				case termbox.KeyCtrlQ:
					return


				case termbox.KeyInsert:	//KeyArrowUp rune is picked up
					ic.Scroll(-1)
				case termbox.KeyDelete: //KeyArrowDown rune is picked up
					ic.Scroll(1)
				case termbox.KeyPgup:
					_,height := termbox.Size()
					ic.Scroll(-height)
				case termbox.KeyPgdn:
					_,height := termbox.Size()
					ic.Scroll(height)

				case termbox.KeyHome:
					ic.Scroll(-1000000000)//-1GB, should exceed msg number
				case termbox.KeyEnd:
					ic.Scroll(1000000000)//1GB, should exceed msg number
				default:
				}
			}
		}
		ic.updateView<-true
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
			ch,ok := ic.Channels[params[0]]
			if !ok {
				ch.Msgs = append(ch.Msgs,params[0])
				ch.Frame_offset += 1
			}
			ch.important = false
			ic.Channels[params[0]] = ch
		ic.map_lock<-true
		ic.current = params[0]
		break
	case "part":
		if len(params) == 0 { break }
		<-ic.map_lock
		_,ok := ic.Channels[params[0]]
		ic.map_lock<-true

		if ok {
			ic.NextChan()
			<-ic.map_lock
			delete(ic.Channels,params[0])
			ic.map_lock<-true
		}
		break
	case "msg":
		pkt.cmd = "privmsg"
		<-ic.map_lock
			ch,_ := ic.Channels[params[0]]
			ch.Msgs = append(ch.Msgs,line)
			ch.Frame_offset += 1
			ic.Channels[params[0]] = ch
		ic.map_lock<-true
	case "privmsg":
		pkt.cmd = "privmsg"
		<-ic.map_lock
			ch,_ := ic.Channels[params[0]]
			ch.Msgs = append(ch.Msgs,line)
			ch.Frame_offset += 1
			ic.Channels[params[0]] = ch
		ic.map_lock<-true
		break
	case "nick":
		ic.Name = strings.Fields(pkt.params)[0]
	case "quit":
		return
	case "":
		<-ic.map_lock
			ch,_ := ic.Channels[ic.current]
			ch.Msgs = append(ch.Msgs,line)
			ch.Frame_offset += 1
			ic.Channels[ic.current] = ch
		ic.map_lock<-true
		pkt.cmd = "privmsg"
		pkt.params = ic.current
		pkt.trail = line
		break
	case "next":
		ic.NextChan()

		//fmt.Println(ic.current)
		//ic.updateView <- true
		return
	case "show":
		<-ic.map_lock
		ch,ok := ic.Channels[params[0]]
		if ok {
			ic.current = params[0]
			ch.important = false
			ic.Channels[params[0]] = ch
		}
		ic.map_lock <- true
		return
	case "l":	//Print channel list
		list := []string{"List of Channels."}
		<-ic.map_lock
		for k,_ := range ic.Channels {
			list = append(list,k+"\t\t"+strconv.Itoa(len(k)))
		}
		ch,_ := ic.Channels["list"]
		ch.Msgs = list
		ch.Frame_offset = len(ch.Msgs)-1
		ic.Channels["list"] = ch
		ic.map_lock <- true
		ic.current = "list"
		return
	case "u":	//Print users to default
		width,height := termbox.Size()
		<-ic.map_lock
			ch,_ := ic.Channels[ic.current]
		ic.map_lock <- true
		out := strings.Join(ch.Users," ")

		block_len := height*width/2
		blocks := (len(out)/block_len)+1

		for i := 0; i<blocks; i++ {
			if (i+1)*block_len > len(out) {
				ic.AddMsg("default",out[i*block_len:])
			} else {
				ic.AddMsg("default",out[i*block_len:(i+1)*block_len])
			}
		}
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
		help = append(help,"/next or CtrlN :Next window.")
		help = append(help,"/find keyword  :Adds keyword to list and highlights sentences.")
		help = append(help,"/clear         :Clears keyword list")
		help = append(help,"CtrlW          :Closes and parts current channel.")
		help = append(help,"Page Up        :Page Up")
		help = append(help,"Page Down      :Page Down")
		help = append(help,"Insert         :Scroll Up")
		help = append(help,"Delete         :Scroll Down")
		<-ic.map_lock
		ch,_ := ic.Channels["help"]
		ch.Msgs = help
		ch.Frame_offset = len(ch.Msgs)-1
		ic.Channels["help"] = ch
		ic.map_lock <- true
		ic.current = "help"
		return
	case "log":
		if len(params) > 0 {
			for _,c := range params {
				ic.Log(c)
			}
		} else {
			var tnames []string
			<-ic.map_lock
			for k,_ := range ic.Channels { tnames = append(tnames,k) }
			ic.map_lock <- true
			for _,n := range tnames { ic.Log(n) }
		}
		return
	case "find":
		ic.keywords = append(ic.keywords,pkt.params)
		return
	case "clear":
		ic.keywords = []string{}
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






