package server

import (
	"context"
	"io"
	"net"
	"net/http"
	"sync"

	"github.com/posener/h2conn"
	"go.uber.org/zap"
)

type Plugin struct {
	Logger   *zap.Logger
	FromAddr string
	ToAddr   string
	Path     string

	KeyFile  string
	CertFile string

	srv *http.Server
}

func (p *Plugin) Serve(ctx context.Context) (err error) {
	mux := http.NewServeMux()
	mux.Handle(p.Path, p)
	srv := &http.Server{Addr: p.FromAddr, Handler: mux}
	p.srv = srv

	if p.KeyFile != "" && p.CertFile != "" {
		return srv.ListenAndServeTLS(p.CertFile, p.KeyFile)
	}

	return srv.ListenAndServe()
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

func (p *Plugin) Close() {
	p.srv.Close()
}
