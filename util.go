package main

import (
	"crypto/aes"
	"crypto/cipher"
	"fmt"
	"hash/fnv"
	"os"
)

// convert the ipaddrs that represents ipv4 address using ipv6
// formats to ipv4 address. otherwise will return unchanged.
func convipaddr624(in []byte) []byte {
	if len(in) != 16 {
		return in
	}
	// check is it can be correctly converted.
	correction := []byte{0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0xFF, 0xFF}
	for i := 0; i < len(correction); i++ {
		if in[i] != correction[i] {
			return in
		}
	}
	// truncate it into ipv4
	return in[12:16]
}
func iappender(a []byte, b []byte) []byte {
	for i := 0; i < len(b); i++ {
		a = append(a, b[i])
	}
	return a
}
func fullbarrcmp(a []byte, b []byte) bool {
	if len(a) != len(b) {
		return false
	}
	for i := 0; i < len(a); i++ {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}
func frntbarrcmp(a []byte, b []byte, n int) bool {
	for i := 0; i < n; i++ {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

func AESEncrypt(barr []byte) []byte {
	var commIV = []byte{0x00, 0x01, 0x02, 0x03, 0x04, 0x05, 0x06, 0x07, 0x08, 0x09, 0x0a, 0x0b, 0x0c, 0x0d, 0x0e, 0x0f}
	key := "samplekeytomatch"
	acipher, cerr := aes.NewCipher([]byte(key))
	if cerr != nil {
		fmt.Printf("Cipher Error.Terminating.\n")
		os.Exit(-1)
	}
	cfb := cipher.NewCFBEncrypter(acipher, commIV)
	encres := make([]byte, len(barr))
	cfb.XORKeyStream(encres, barr)
	return encres
}

func AESDecrypt(barr []byte) []byte {
	var commIV = []byte{0x00, 0x01, 0x02, 0x03, 0x04, 0x05, 0x06, 0x07, 0x08, 0x09, 0x0a, 0x0b, 0x0c, 0x0d, 0x0e, 0x0f}
	key := "samplekeytomatch"
	acipher, cerr := aes.NewCipher([]byte(key))
	if cerr != nil {
		fmt.Printf("Cipher Error.Terminating.\n")
		os.Exit(-1)
	}
	cfb := cipher.NewCFBDecrypter(acipher, commIV)
	decres := make([]byte, len(barr))
	cfb.XORKeyStream(decres, barr)
	return decres
}

func addrxport2id(ipaddr []byte, port []byte) uint32 {
	tmpbarr := iappender(ipaddr, port)
	h := fnv.New32a()
	h.Write(tmpbarr)
	return h.Sum32()
}
