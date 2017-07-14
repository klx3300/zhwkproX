package main

import (
	"encoding/binary"
	"fmt"
	"net"
	"os"
	"time"
)

type IOPair struct {
	in  chan []byte
	out chan []byte
}

const (
	SRVSTATE_ERR = iota
)

func main() {
	server := ":1992"
	stcpaddr, _ := net.ResolveTCPAddr("tcp4", server)
	conn, cerr := net.DialTCP("tcp4", nil, stcpaddr)
	if cerr != nil {
		fmt.Printf("Network error.Terminating..\n")
		fmt.Println(cerr.Error())
		return
	}
	defer conn.Close()
	// hs
	var hsm zhwkHsMsg
	hsm.version = 0x80
	hsm.numMethods = 0x01
	hsm.methodarr = []byte{0x00}
	conn.Write(hsm.toByteArr())
	buffbyte := make([]byte, 1)
	_, err := conn.Read(buffbyte)
	if err != nil {
		fmt.Printf("Network Error.Terminating..\n")
		return
	}
	if buffbyte[0] != 0x80 {
		fmt.Printf("Protocol implementation error.Terminating...\n")
		return
	}
	// auth
	authstr := "fake random str.123456"
	var zar zhwkAuthReply
	zar.repsize = byte(len(authstr))
	zar.encmsg = []byte(authstr)
	conn.Write(zar.toByteArr())
	fmt.Printf("Waiting Authentication..\n")
	buffbyte = make([]byte, 1024)
	n, err := conn.Read(buffbyte)
	fmt.Printf("Authenticating..\n")
	if err != nil {
		fmt.Printf("Network Error.Terminating...\n")
		return
	}
	buffbyte = buffbyte[:n]
	if !frntbarrcmp(AESDecrypt(buffbyte[1:]), []byte(authstr), len(authstr)) {
		fmt.Printf("Authentication Error.Terminating..\n")
		return
	}
	fmt.Printf("Authentication Successive..\n")
	buffbyte = make([]byte, 1)
	buffbyte[0] = 0x01
	_, err = conn.Write(buffbyte)
	if err != nil {
		fmt.Printf("Network Error.Terminated..\n")
		return
	}
	// connection established.
	// create a map object to storage tcp connections
	conns := make(map[uint32]IOPair)
	for {
		// accept requests from..
		buffbyte = make([]byte, 1)
		var zgr zhwkGetRequest
		_, err := conn.Read(buffbyte)
		handleErr := func(err error) {
			if err != nil {
				fmt.Printf("Network Error %v.Terminating..\n", err.Error())
				os.Exit(-1)
			}
		}
		handleErr(err)
		zgr.ipversion = buffbyte[0]
		// read ipaddr
		switch zgr.ipversion {
		case 0x04:
			zgr.ipaddr = make([]byte, 4)
			_, err = conn.Read(zgr.ipaddr)
			handleErr(err)
		case 0x06:
			zgr.ipaddr = make([]byte, 16)
			_, err = conn.Read(zgr.ipaddr)
			handleErr(err)
		}
		// read port
		zgr.port = make([]byte, 2)
		_, err = conn.Read(zgr.port)
		handleErr(err)
		tmpdtlen := make([]byte, 4)
		_, err = conn.Read(tmpdtlen)
		handleErr(err)
		zgr.datalength = binary.LittleEndian.Uint32(tmpdtlen)
		if zgr.datalength != 0 {
			zgr.data = make([]byte, int(zgr.datalength))
			_, err = conn.Read(zgr.data)
			handleErr(err)
		}
		go func(zgr zhwkGetRequest) {
			zgr.data = AESDecrypt(zgr.data)
			pair, isok := conns[addrxport2id(zgr.ipaddr, zgr.port)]
			if !isok {
				// havent created yet.
				var tmpiop IOPair
				tmpiop.in = make(chan []byte, 5)
				tmpiop.out = make(chan []byte, 5)
				// start a goroutine to listen at this iopair and operation throughout network
				go func() {
					// open tcp connection
					var dsttcpaddr net.TCPAddr
					dsttcpaddr.IP = zgr.ipaddr
					fmt.Println(zgr)
					dsttcpaddr.Port = int(binary.BigEndian.Uint16(zgr.port))
					// startup tcp connection
					sconn, err := net.DialTCP("tcp", nil, &dsttcpaddr)
					if err != nil {
						fmt.Printf("NetworkError.Cancelling...\n")
						fmt.Println(err.Error())
						delete(conns, addrxport2id(zgr.ipaddr, zgr.port))
						return
					}
					defer func() {
						// close gochannels
						delete(conns, addrxport2id(zgr.ipaddr, zgr.port))
						sconn.Close()
					}()
					// start a func to receive those inputs
					go func() {
						for {
							tmpbuffer := make([]byte, 102400)
							n, err := sconn.Read(tmpbuffer)
							if err != nil {
								fmt.Printf("Prox to Dest Conn error %v. Terminating..\n", err.Error())
								close(tmpiop.out)
								delete(conns, addrxport2id(zgr.ipaddr, zgr.port))
								sconn.Close()
								return
							}
							tmpbuffer = tmpbuffer[:n]
							tmpiop.out <- tmpbuffer
						}
					}()
					for {
						select {
						case data := <-tmpiop.in:
							_, err := sconn.Write(data)
							if err != nil {
								fmt.Printf("Network Error.Retrying(In case http closure)..\n")
								sconn, err = net.DialTCP("tcp", nil, &dsttcpaddr)
								if err != nil {
									fmt.Printf("Network fail retry failed.Terminating..\n")
									return
								}
								defer sconn.Close()
								_, err := sconn.Write(data)
								if err != nil {
									fmt.Printf("Network fail retry failed.Terminating..\n")
									return
								}
							}
						case <-time.After(20 * time.Second):
							fmt.Printf("Timed out.Terminating..\n")
							return
						}
					}
				}()
				conns[addrxport2id(zgr.ipaddr, zgr.port)] = tmpiop
				pair = tmpiop
				// start a goroutine to automatically write output out.
				go func() {
					for {
						tmpbuff, ok := <-pair.out
						if !ok {
							// constuct zgrr
							var zgrr zhwkGetReply
							zgrr.ipversion = zgr.ipversion
							zgrr.ipaddr = zgr.ipaddr
							zgrr.port = zgr.port
							zgrr.datalength = 0x00
							zgrr.data = make([]byte, 1)
							_, err := conn.Write(zgrr.toByteArr())
							if err != nil {
								fmt.Printf("Interserver side Conn error %v.Terminating..\n", err.Error())
								conn.Close()
								os.Exit(-1)
							}
							return
						}
						fmt.Printf("Received str:\n%v\n", string(tmpbuff))
						// construct reply
						var zgrr zhwkGetReply
						zgrr.ipversion = zgr.ipversion
						zgrr.ipaddr = zgr.ipaddr
						zgrr.port = zgr.port
						zgrr.datalength = uint32(len(tmpbuff))
						zgrr.data = AESEncrypt(tmpbuff)
						// send it out
						go func() {
							_, err := conn.Write(zgrr.toByteArr())
							if err != nil {
								fmt.Printf("Network error.Terminating..\n")
								conn.Close()
								os.Exit(-1)
							}
						}()
					}
				}()
			}
			// write data into..
			pair.in <- zgr.data
		}(zgr)
	}
}
