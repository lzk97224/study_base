package main

import (
	"log"
	"net"
	"testing"
)

func TestTCP(t *testing.T) {
	listen, err := net.Listen("tcp", ":8888")
	if err != nil {
		log.Panic("监听失败")
	}
	conn, err := listen.Accept()
	if err != nil {
		log.Panic(err)
	}

	var body [100]byte
	_, err = conn.Read(body[:])

	_, err = conn.Write(body[:])

}
