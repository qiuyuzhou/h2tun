package server

import (
	"context"
	"io"
	"net"
	"net/http"
	"sync"
	"time"

	"github.com/posener/h2conn"
	"go.uber.org/zap"
	"golang.org/x/net/http2"
	"golang.org/x/net/http2/h2c"
)

type Plugin struct {
	Logger   *zap.Logger
	FromAddr string
	ToAddr   string
	Path     string

	KeyFile  string
	CertFile string
}

func (p *Plugin) Serve(ctx context.Context) (err error) {
	serveTLS := (p.KeyFile != "" && p.CertFile != "")

	p.Logger.Info(
		"Start serving...",
		zap.String("fromAddr", p.FromAddr),
		zap.String("toAddr", p.ToAddr),
		zap.String("path", p.Path),
		zap.String("keyFile", p.KeyFile),
		zap.String("certFile", p.CertFile),
		zap.Bool("serveTLS", serveTLS),
	)

	mux := http.NewServeMux()
	mux.Handle(p.Path, p)

	var handler http.Handler

	if serveTLS {
		handler = mux
	} else {
		// Enable h2c
		h2s := &http2.Server{}
		handler = h2c.NewHandler(mux, h2s)
	}

	srv := &http.Server{
		Addr:    p.FromAddr,
		Handler: handler,
	}

	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()
		select {
		case <-ctx.Done():
			ctx, cancel := context.WithTimeout(context.Background(), time.Second)
			defer cancel()
			srv.Shutdown(ctx)
		}
	}()

	if serveTLS {
		err = srv.ListenAndServeTLS(p.CertFile, p.KeyFile)
	} else {
		err = srv.ListenAndServe()
	}

	wg.Wait()

	return
}

func (p *Plugin) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	fromConn, err := h2conn.Accept(w, r)
	if err != nil {
		p.Logger.Sugar().Warnf("Failed creating full duplex connection from %s: %s", r.RemoteAddr, err)
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}

	defer fromConn.Close()

	toConn, err := net.Dial("tcp", p.ToAddr)
	if err != nil {
		p.Logger.Sugar().Warnf("Failed creating connection to %s: %s", p.ToAddr, err)
	}
	defer toConn.Close()

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
