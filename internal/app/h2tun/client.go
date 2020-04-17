package h2tun

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

type Client struct {
	Logger             *zap.Logger
	FromAddr           string
	ToAddr             string
	Path               string
	UseTLS             bool
	InsecureSkipVerify bool

	toURL      string
	httpClient *h2conn.Client
}

func (p *Client) Serve(ctx context.Context) (err error) {
	if p.UseTLS {
		p.toURL = fmt.Sprintf("https://%s%s", p.ToAddr, p.Path)
	} else {
		p.toURL = fmt.Sprintf("http://%s%s", p.ToAddr, p.Path)
	}

	p.Logger.Info(
		"Start serving...",
		zap.String("fromAddr", p.FromAddr),
		zap.String("toAddr", p.ToAddr),
		zap.String("path", p.Path),
		zap.Bool("useTLS", p.UseTLS),
		zap.String("toURL", p.toURL),
	)

	var transport *http2.Transport
	if p.UseTLS {
		transport = &http2.Transport{
			TLSClientConfig: &tls.Config{
				InsecureSkipVerify: p.InsecureSkipVerify,
			},
		}
	} else {
		transport = &http2.Transport{
			AllowHTTP: true,
			// Workaround to get the golang standard http2 client to connect to an H2C enabled server
			DialTLS: func(network, addr string, cfg *tls.Config) (net.Conn, error) {
				return net.Dial(network, addr)
			},
		}
	}

	client := &h2conn.Client{
		Client: &http.Client{
			Transport: transport,
		},
	}
	p.httpClient = client

	ln, err := net.Listen("tcp", p.FromAddr)
	if err != nil {
		p.Logger.Sugar().Fatalf("Failed to listen on the addr: %s", p.FromAddr)
	}

	go func() {
		select {
		case <-ctx.Done():
			ln.Close()
		}
	}()

	for {
		conn, _err := ln.Accept()
		if _err != nil {
			return
		}
		go p.handleConn(ctx, conn)
	}
}

func (p *Client) handleConn(ctx context.Context, fromConn net.Conn) {
	defer fromConn.Close()

	toConn, resp, err := p.httpClient.Connect(ctx, p.toURL)
	if err != nil {
		p.Logger.Warn(
			"Failed to connect to server",
			zap.String("url", p.toURL),
			zap.Error(err),
		)
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
		toConn.Close()
	}()
	go func() {
		io.Copy(toConn, fromConn)
		fromConn.Close()
	}()

	wg.Wait()
}
