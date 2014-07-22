package main

import (
	"strings"
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
		pkt.prefix = strings.ToLower(split[0][1:])
	} else {
		pkt.prefix = ""
	}
	pkt.cmd = strings.ToLower(split[off])
	pkt.params = strings.Join(split[off+1:]," ")
	return pkt
}

