package main

import (
	"fmt"
	"github.com/go-yaml/yaml"
	"github.com/golang/glog"
	"io"
	"io/ioutil"
	//"crypto/tls"
	"flag"
	//"log"
	"net"
)

func getSNIServerName(buf []byte) string {
	n := len(buf)
	if n < 5 {
		glog.Error("not tls handshake")
		return ""
	}

	// tls record type
	if recordType(buf[0]) != recordTypeHandshake {
		glog.Error("not tls")
		return ""
	}

	// tls major version
	if buf[1] != 3 {
		glog.Error("TLS version < 3 not supported")
		return ""
	}

	// payload length
	//l := int(buf[3])<<16 + int(buf[4])

	//log.Printf("length: %d, got: %d", l, n)

	// handshake message type
	if uint8(buf[5]) != typeClientHello {
		glog.Error("not client hello")
		return ""
	}

	// parse client hello message

	msg := &clientHelloMsg{}

	// client hello message not include tls header, 5 bytes
	ret := msg.unmarshal(buf[5:n])
	if !ret {
		glog.Error("parse hello message return false")
		return ""
	}
	return msg.serverName
}

func forward(c net.Conn, data []byte, dst string) {
	c1, err := net.Dial("tcp", dst)
	if err != nil {
		glog.Error(err)
		return
	}

	defer c1.Close()

	if _, err = c1.Write(data); err != nil {
		glog.Error(err)
		return
	}

	ch := make(chan struct{}, 2)

	go func() {
		io.Copy(c1, c)
		ch <- struct{}{}
	}()

	go func() {
		io.Copy(c, c1)
		ch <- struct{}{}
	}()

	<-ch
}

func getDST(c net.Conn, name string) string {
	addr := c.LocalAddr().(*net.TCPAddr)
	dst := cfg.ForwardRules.GetN(name, addr.Port)
	return dst
}

func getDefaultDST() string {
	return cfg.Default
}

func serve(c net.Conn) {
	defer c.Close()

	buf := make([]byte, 1024)
	n, err := c.Read(buf)
	if err != nil {
		glog.Error(err)
		return
	}
	servername := getSNIServerName(buf[:n])
	if servername == "" {
		forward(c, buf[:n], getDefaultDST())
		return
	}
	dst := getDST(c, servername)
	if dst == "" {
		dst = getDefaultDST()
	}
	forward(c, buf[:n], dst)
}

var cfg conf

func main() {
	var cfgfile string
	flag.StringVar(&cfgfile, "c", "config.yaml", "config file")
	flag.Set("logtostderr", "true")
	flag.Parse()

	data, err := ioutil.ReadFile(cfgfile)
	if err != nil {
		glog.Fatal(err)
	}
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		glog.Fatal(err)
	}

	for _, d := range cfg.Listen {
		glog.Infof("listen on :%d", d)
		l, err := net.Listen("tcp", fmt.Sprintf(":%d", d))
		if err != nil {
			glog.Fatal(err)
		}
		go func(l net.Listener) {
			defer l.Close()
			for {
				c1, err := l.Accept()
				if err != nil {
					glog.Fatal(err)
				}
				go serve(c1)
			}
		}(l)
	}
	select {}
}
