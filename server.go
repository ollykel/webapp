package webapp

/**
 * @author Oliver Kelton, oakelton@gmail.com
 * @date Mar 20, 2019
 * Wrapper for net/http Server struct, to satisfy Server interface
 */

import (
	"net"
	"net/http"
	"context"
)

type Server interface {
	Close () error
	ListenAndServe () error
	ListenAndServeTLS (certFile, keyFile string) error
	Serve (l net.Listener) error
	ServeTLS (l net.Listener, certFile, keyFile string) error
	Shutdown (ctx context.Context) error
	GetAddr () string
	SetAddr (addr string)
	SetHandler (handler Handler)
}//-- end Server interface

// Wrapper for default net/http Server, to satisfy interface
type DefaultServer struct {
	http.Server
}//-- end DefaultServer struct

func (svr *DefaultServer) GetAddr () string {
	return svr.Addr
}//-- end func DefaultServer.GetAddr

func (svr *DefaultServer) SetAddr (addr string) {
	svr.Addr = addr
}//-- end func DefaultServer.SetAddr

func (svr *DefaultServer) SetHandler (handler Handler) {
	svr.Handler = handler
}//-- end DefaultServer.SetHandler

