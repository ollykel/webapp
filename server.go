package webapp

/**
 * @author Oliver Kelton, oakelton@gmail.com
 * @date Mar 20, 2019
 * Wrapper for net/http Server struct, to satisfy Server interface
 */

import (
	"log"
	"fmt"
	"net/http"
	"context"
	"errors"
)

type ServerConfig struct {
	Port string
	StaticDir string
	TLSEnabled bool
	CertFile, KeyFile string
	CacheTimeoutSecs int
}//-- end ServerConfig struct

func (cfg *ServerConfig) Validate () error {
	if cfg.TLSEnabled && (cfg.CertFile == "" || cfg.KeyFile == "") {
		return errors.New("TLSEnabled, but CertFile or KeyFile not given")
	}
	return nil
}//-- end DefaultServer.Validate

type Server interface {
	Init (cfg *ServerConfig, handler Handler) error
	Close () error
	Serve () error
	ServeStatic (w http.ResponseWriter, r *http.Request)
	Shutdown (ctx context.Context) error
	GetAddr () string
}//-- end Server interface

// Wrapper for default net/http Server, to satisfy interface
type DefaultServer struct {
	http.Server
	staticServer cachedStaticServer
	tlsEnabled bool
	certFile, keyFile string
}//-- end DefaultServer struct

func (svr *DefaultServer) Init (cfg *ServerConfig, handler Handler) error {
	if cfg == nil { return errors.New("passed nil ServerConfig") }
	if handler == nil { return errors.New("passed nil Handler") }
	err := cfg.Validate()
	if err != nil { return err }
	svr.Addr = cfg.Port
	svr.Handler = handler
	svr.staticServer = makeStaticServer(cfg)
	svr.tlsEnabled = cfg.TLSEnabled
	if cfg.TLSEnabled {
		svr.certFile, svr.keyFile = cfg.CertFile, cfg.KeyFile
	}
	return nil
}//-- end func DefaultServer.Init

func (svr *DefaultServer) GetAddr () string {
	return svr.Addr
}//-- end func DefaultServer.GetAddr

func (svr *DefaultServer) Serve () error {
	if svr.tlsEnabled {
		return svr.ListenAndServeTLS(svr.certFile, svr.keyFile)
	}
	return svr.ListenAndServe()
}//-- end func DefaultServer.ListenAndServe

type cachedStaticServer func (w http.ResponseWriter, r *http.Request)

func makeStaticServer (cfg *ServerConfig) cachedStaticServer {
	cacheHeader := fmt.Sprintf("max-age=%d", cfg.CacheTimeoutSecs)
	return func (w http.ResponseWriter, r *http.Request) {
		log.Printf("serveStatic: %s\n", r.URL.Path)
		var filename string
		if r.URL.Path == "/" {
			filename = cfg.StaticDir + "/index.html"
		} else {
			filename = cfg.StaticDir + r.URL.Path
			w.Header().Set("Cache-Control", cacheHeader)
		}
		// allows caching for optimized performance
		//-- particularly important for js bundles
		http.ServeFile(w, r, filename)
	}//-- end return
}//-- end func makeStaticServer

func (svr *DefaultServer) ServeStatic (w http.ResponseWriter,
		r *http.Request) {
	svr.staticServer(w, r)
}//-- end Webapp.serveStatic

