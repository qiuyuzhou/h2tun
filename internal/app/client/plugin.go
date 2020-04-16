package client

import (
	"context"
	"crypto/tls"
	"fmt"
	"io"
	"net"
	"net/http"
	"sync"

	"github.com/posener/h2conn"
	"go.uber.org/zap"
	"golang.org/x/net/http2"
)

type Plugin struct {
	Logger             *zap.Logger
	FromAddr           string
	ToAddr             string
	Path               string
	UseTLS             bool
	InsecureSkipVerify bool

	toURL      string
	listener   net.Listener
	httpClient *h2conn.Client
}

func (p *Plugin) Serve(ctx context.Context) (err error) {
	if p.UseTLS {
		p.toURL = fmt.Sprintf("https://%s%s", p.ToAddr, p.Path)
	} else {
		p.toURL = fmt.Sprintf("http://%s%s", p.ToAddr, p.Path)
	}

	client := &h2conn.Client{
		Client: &http.Client{
			Transport: &http2.Transport{TLSClientConfig: &tls.Config{
				InsecureSkipVerify: p.InsecureSkipVerify,
			}},
		},
	}
	p.httpClient = client

	ln, err := net.Listen("tcp", p.FromAddr)
	if err != nil {
		p.Logger.Sugar().Fatalf("Failed to listen on the addr: %s", p.FromAddr)
	}
	p.listener = ln

	for {
		conn, err := ln.Accept()
		if err != nil {
			p.Logger.Sugar().Fatalf("Failed to accept connection from addr: %s", p.FromAddr)
		}
		go p.handleConn(ctx, conn)
	}
}

func (p *Plugin) handleConn(ctx context.Context, fromConn net.Conn) {
	defer fromConn.Close()

	toConn, resp, err := p.httpClient.Connect(ctx, p.toURL)
	if err != nil {
		p.Logger.Sugar().Warnf("Failed to connect to: %s", p.toURL)
		return
	}
	defer toConn.Close()

	if resp.StatusCode != http.StatusOK {
		p.Logger.Sugar().Warnf("http2 server return status: %s", resp.Status)
		return
	}

	var wg sync.WaitGroup
	wg.Add(2)
	go func() {
		io.Copy(fromConn, toConn)
	}()
	go func() {
		io.Copy(toConn, fromConn)
	}()

	wg.Wait()
}

func (p *Plugin) Close() {
	p.listener.Close()
}
