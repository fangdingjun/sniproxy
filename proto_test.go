package main

import (
	"context"
	"crypto/tls"
	"io/ioutil"
	"net"
	"testing"

	"github.com/fangdingjun/go-log"
	"github.com/fangdingjun/protolistener"
	yaml "gopkg.in/yaml.v2"
)

func TestProxyProto(t *testing.T) {
	log.Default.Level = log.DEBUG

	data, err := ioutil.ReadFile("config.sample.yaml")
	if err != nil {
		log.Fatal(err)
	}
	if err = yaml.Unmarshal(data, &cfg); err != nil {
		log.Fatal(err)
	}

	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	defer listener.Close()
	log.Printf("listen %s", listener.Addr().String())

	go func() {
		for {
			conn, err := listener.Accept()
			if err != nil {
				return
			}
			go serve(context.Background(), conn)
		}
	}()
	cert, err := tls.LoadX509KeyPair("server.crt", "server.key")
	if err != nil {
		t.Fatal(err)
	}
	listener2, err := net.Listen("tcp", "127.0.0.1:8443")
	if err != nil {
		t.Fatal(err)
	}
	defer listener2.Close()

	listener2 = tls.NewListener(protolistener.New(listener2), &tls.Config{
		Certificates: []tls.Certificate{cert},
	})

	go func() {
		for {
			conn, err := listener2.Accept()
			if err != nil {
				return
			}
			go func(c net.Conn) {
				defer c.Close()
				addr := conn.RemoteAddr()
				_conn := c.(*tls.Conn)
				if err := _conn.Handshake(); err != nil {
					log.Errorf("handshake error: %s", err)
					return
				}
				conn.Write([]byte(addr.String()))
			}(conn)
		}
	}()

	conn, err := tls.Dial("tcp", listener.Addr().String(), &tls.Config{
		ServerName:         "www.example.com",
		InsecureSkipVerify: true,
	})
	if err != nil {
		log.Println("dial error")
		t.Fatal(err)
	}
	conn.Handshake()
	buf := make([]byte, 200)
	n, err := conn.Read(buf)
	if err != nil {
		log.Println("read error")
		t.Fatal(err)
	}
	addr1 := conn.LocalAddr().String()
	addr2 := string(buf[:n])
	conn.Close()
	if addr1 != addr2 {
		t.Errorf("expect %s, got: %s", addr1, addr2)
	}
}
