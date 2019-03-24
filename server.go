package webapp

/**
 * @author Oliver Kelton, oakelton@gmail.com
 * @date Mar 20, 2019
 * Wrapper for net/http Server struct, to satisfy Server interface
 */

import (
	"net/http"
	"context"
	"fmt"
	"log"
	"errors"
	"time"
)

type ServerConfig struct {
	Port string
	WaitSecs int
	TCPEnabled bool
	CertFile, KeyFile string
}//-- end ServerConfig struct

func (cfg *ServerConfig) Validate () error {
	if cfg.WaitSecs < 0 {
		return fmt.Errorf("WaitSecs must be 0 or greater (got: %d)",
			cfg.WaitSecs)
	}
	if cfg.TCPEnabled && (cfg.CertFile == "" || cfg.KeyFile == "") {
		return errors.New("TCPEnabled, but CertFile or KeyFile not given")
	}
	return nil
}//-- end DefaultServer.Validate

type Server interface {
	Init (cfg *ServerConfig, handler Handler) error
	Close () error
	Serve () error
	Shutdown (ctx context.Context) error
	GetAddr () string
}//-- end Server interface

// Wrapper for default net/http Server, to satisfy interface
type DefaultServer struct {
	http.Server
	waitSecs int
	tcpEnabled bool
	certFile, keyFile string
}//-- end DefaultServer struct

func (svr *DefaultServer) Init (cfg *ServerConfig, handler Handler) error {
	if cfg == nil { return errors.New("passed nil ServerConfig") }
	if handler == nil { return errors.New("passed nil Handler") }
	err := cfg.Validate()
	if err != nil { return err }
	svr.Addr = cfg.Port
	svr.Handler = handler
	svr.waitSecs = cfg.WaitSecs
	svr.tcpEnabled = cfg.TCPEnabled
	if cfg.TCPEnabled {
		svr.certFile, svr.keyFile = cfg.CertFile, cfg.KeyFile
	}
	return nil
}//-- end func DefaultServer.Init

func (svr *DefaultServer) GetAddr () string {
	return svr.Addr
}//-- end func DefaultServer.GetAddr

func (svr *DefaultServer) Serve () error {
	if svr.waitSecs > 0 {
		log.Printf("Waiting %d seconds...", svr.waitSecs)
		time.Sleep(time.Duration(svr.waitSecs) * time.Second)
	}
	if svr.tcpEnabled {
		return svr.ListenAndServeTLS(svr.certFile, svr.keyFile)
	}
	return svr.ListenAndServe()
}//-- end func DefaultServer.ListenAndServe

