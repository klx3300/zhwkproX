package main

import (
	"encoding/binary"
	"fmt"
)

type zhwkHsMsg struct {
	version    byte
	numMethods byte
	methodarr  []byte
}

func (zhm *zhwkHsMsg) toByteArr() []byte {
	barr := make([]byte, 2)
	barr[0] = zhm.version
	barr[1] = zhm.numMethods
	barr = iappender(barr, zhm.methodarr)
	return barr
}

type zhwkAuthMsg struct {
	msgsize byte
	msg     []byte
}

func makeAuthMsg(barr []byte) zhwkAuthMsg {
	var am zhwkAuthMsg
	am.msgsize = barr[0]
	am.msg = barr[1 : 1+am.msgsize]
	return am
}

type zhwkAuthReply struct {
	repsize byte
	encmsg  []byte
}

func (zar *zhwkAuthReply) toByteArr() []byte {
	barr := make([]byte, 1)
	barr[0] = zar.repsize
	for it := 0; it < len(zar.encmsg); it++ {
		barr = append(barr, zar.encmsg[it])
	}
	return barr
}

type zhwkGetReply struct {
	ipversion  byte
	ipaddr     []byte
	port       []byte
	datalength uint32
	data       []byte
}

func (zgr *zhwkGetReply) toByteArr() []byte {
	barr := make([]byte, 1)
	barr[0] = zgr.ipversion
	barr = iappender(barr, zgr.ipaddr)
	barr = iappender(barr, zgr.port)
	tmpcnv := make([]byte, 4)
	binary.LittleEndian.PutUint32(tmpcnv, zgr.datalength)
	barr = iappender(barr, tmpcnv)
	barr = iappender(barr, zgr.data)
	return barr
}

type zhwkGetRequest struct {
	ipversion  byte
	ipaddr     []byte
	port       []byte
	datalength uint32
	data       []byte
}

func makeGetRequest(barr []byte) zhwkGetRequest {
	fmt.Println(barr)
	var tmp zhwkGetRequest
	tmp.ipversion = barr[0]
	switch tmp.ipversion {
	case 0x04:
		tmp.ipaddr = barr[1:5]
		tmp.port = barr[5:7]
		tmp.datalength = binary.LittleEndian.Uint32(barr[7:11])
		tmp.data = barr[11:]
	case 0x06:
		tmp.ipaddr = barr[1:17]
		tmp.port = barr[17:19]
		tmp.datalength = binary.LittleEndian.Uint32(barr[19:23])
		tmp.data = barr[23:]
	}
	fmt.Println(tmp)
	return tmp
}
