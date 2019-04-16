package main

import (
	"context"
	"flag"
	"fmt"
	"io"
	"io/ioutil"
	"net"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	glog "github.com/fangdingjun/go-log"
	proxyproto "github.com/pires/go-proxyproto"
	yaml "gopkg.in/yaml.v2"
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

func forward(ctx context.Context, c net.Conn, data []byte, dst string) {
	addr := dst
	proxyProto := 0

	ss := strings.Fields(dst)

	var hdr proxyproto.Header

	if len(ss) > 1 {
		addr = ss[0]
		raddr := c.RemoteAddr().(*net.TCPAddr)
		glog.Debugf("connection from %s", raddr)
		hdr = proxyproto.Header{
			Version:            1,
			Command:            proxyproto.PROXY,
			TransportProtocol:  proxyproto.TCPv4,
			SourceAddress:      raddr.IP.To4(),
			DestinationAddress: net.IP{0, 0, 0, 0},
			SourcePort:         uint16(raddr.Port),
			DestinationPort:    0,
		}

		switch strings.ToLower(ss[1]) {
		case "proxy-v1":
			proxyProto = 1
			hdr.Version = 1
		case "proxy-v2":
			proxyProto = 2
			hdr.Version = 2
		}
	}
	dialer := &net.Dialer{Timeout: 10 * time.Second}
	c1, err := dialer.DialContext(ctx, "tcp", addr)
	if err != nil {
		glog.Error(err)
		return
	}

	defer c1.Close()

	if proxyProto != 0 {
		glog.Debugf("send proxy proto v%d to %s", proxyProto, addr)
		if _, err = hdr.WriteTo(c1); err != nil {
			glog.Errorln(err)
			return
		}
	}

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

	select {
	case <-ch:
	case <-ctx.Done():
	}
}

func getDST(c net.Conn, name string) string {
	addr := c.LocalAddr().(*net.TCPAddr)
	dst := cfg.ForwardRules.GetN(name, addr.Port)
	return dst
}

func getDefaultDST() string {
	return cfg.Default
}

func serve(ctx context.Context, c net.Conn) {
	defer c.Close()

	buf := make([]byte, 1024)
	n, err := c.Read(buf)
	if err != nil {
		glog.Error(err)
		return
	}
	servername := getSNIServerName(buf[:n])
	if servername == "" {
		glog.Debugf("no sni, send to default")
		forward(ctx, c, buf[:n], getDefaultDST())
		return
	}
	dst := getDST(c, servername)
	if dst == "" {
		dst = getDefaultDST()
		glog.Debugf("use default dst %s for sni %s", dst, servername)
	}
	forward(ctx, c, buf[:n], dst)
}

var cfg conf

func main() {
	var cfgfile string
	var logfile string
	var loglevel string
	flag.StringVar(&cfgfile, "c", "config.yaml", "config file")
	flag.StringVar(&logfile, "log_file", "", "log file")
	flag.StringVar(&loglevel, "log_level", "INFO", "log level")
	flag.Parse()

	data, err := ioutil.ReadFile(cfgfile)
	if err != nil {
		glog.Fatal(err)
	}
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		glog.Fatal(err)
	}

	if logfile != "" {
		glog.Default.Out = &glog.FixedSizeFileWriter{
			MaxCount: 4,
			Name:     logfile,
			MaxSize:  10 * 1024 * 1024,
		}
	}

	if lv, err := glog.ParseLevel(loglevel); err == nil {
		glog.Default.Level = lv
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	for _, d := range cfg.Listen {
		glog.Infof("listen on :%d", d)
		lc := &net.ListenConfig{}
		l, err := lc.Listen(ctx, "tcp", fmt.Sprintf(":%d", d))
		if err != nil {
			glog.Fatal(err)
		}
		go func(ctx context.Context, l net.Listener) {
			defer l.Close()
			for {
				c1, err := l.Accept()
				if err != nil {
					glog.Error(err)
					break
				}
				go serve(ctx, c1)
			}
		}(ctx, l)
	}

	ch := make(chan os.Signal, 2)
	signal.Notify(ch, syscall.SIGINT, syscall.SIGTERM)
	select {
	case s := <-ch:
		cancel()
		glog.Printf("received signal %s, exit.", s)
	}
}
