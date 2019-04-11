package webapp

/**
 * @author Oliver Kelton, oakelton@gmail.com
 * @date Mar 20, 2019
 * Wrapper for net/http Server struct, to satisfy Server interface
 */

import (
	"log"
	"fmt"
	"os"
	"io/ioutil"
	"net/http"
	"context"
	"errors"
	"sync"
	"time"
)

type ServerConfig struct {
	Port string
	StaticDir string
	TLSEnabled bool
	CertFile, KeyFile string
	CacheTimeoutSecs int
	StaticCacheRefreshSecs int
}//-- end ServerConfig struct

func (cfg *ServerConfig) Validate () error {
	if cfg.Port == "" {
		return errors.New("No Port provided to ServerConfig")
	}
	if cfg.TLSEnabled && (cfg.CertFile == "" || cfg.KeyFile == "") {
		return errors.New("TLSEnabled, but CertFile or KeyFile not given")
	}
	if cfg.StaticDir == "" {
		return errors.New("No StaticDir provided to ServerConfig")
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

type handlerMap struct {
	handlers map[string]http.HandlerFunc
	mut sync.RWMutex
}//-- end handlerMap struct

func (hm *handlerMap) Get (key string) http.HandlerFunc {
	hm.mut.RLock()
	defer hm.mut.RUnlock()
	return hm.handlers[key]
}//-- end func handlerMap.Get

func getFileType (f *os.File, content []byte) string {
	info, err := f.Stat()
	if err != nil { log.Fatal(err.Error()) }
	name := info.Name()
	lastPeriod := strings.LastIndexByte(name, '.')
	if lastPeriod > -1 && lastPeriod < len(name) {
		ext := name[lastPeriod + 1:]
		fileType := fileTypes[ext]//-- see filetypes.go
		if fileType != "" { return fileType }
	}
	return http.DetectContentType(content)
}//-- end func getFileType

func (hm *handlerMap) loadFile (file *os.File, filename string) {
	content, _ := ioutil.ReadAll(file)
	contentType := getFileType(file, content)
	hm.handlers["/" + filename] = func (w http.ResponseWriter,
			_ *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Header().Set("Content-Type", contentType)
		w.Write(content)
	}//-- end func
}//-- end handlerMap.loadFile

func (hm *handlerMap) loadFilesRec (dir *os.File, dirName, prefix string) {
	if prefix != "" { prefix += "/" }
	dirName += "/"
	fileInfos, _ := dir.Readdir(0)//-- ls directory
	var (
		file *os.File
		err error
		filename string
	)
	for _, info := range fileInfos {
		filename = prefix + info.Name()
		file, err = os.Open(dirName + filename)
		if err != nil {
			log.Fatal(err.Error())
			continue
		}
		if info.IsDir() {
			hm.loadFilesRec(file, dirName, filename)
		} else {
			hm.loadFile(file, filename)
		}
		file.Close()
	}//-- end for range fileInfos
}//-- end func handlerMap.loadFilesRec

func (hm *handlerMap) LoadFiles (dir *os.File, dirName string) {
	hm.mut.Lock()
	defer hm.mut.Unlock()
	hm.loadFilesRec(dir, dirName, "")
}//-- end func handlerMap

func (hm *handlerMap) LoadFilesInterval (dirName string,
		interv time.Duration) {
	go func() {
		for {
			staticDir, err := os.Open(dirName)
			defer staticDir.Close()
			if err != nil { log.Fatal(err) }
			hm.LoadFiles(staticDir, dirName)
			if interv < 1 { break }
			time.Sleep(interv)
		}
	}()
}//-- end handlerMap.LoadFilesInterval

func initHandlerMap () (hm handlerMap) {
	hm.handlers = make(map[string]http.HandlerFunc)
	return
}//-- end func initHandlerMap

type cachedStaticServer func (w http.ResponseWriter, r *http.Request)

func makeStaticServer (cfg *ServerConfig) cachedStaticServer {
	handlers := initHandlerMap()
	handlers.LoadFilesInterval(cfg.StaticDir,
		time.Duration(cfg.StaticCacheRefreshSecs) * time.Second)
	cacheHeader := fmt.Sprintf("max-age=%d", cfg.CacheTimeoutSecs)
	return func (w http.ResponseWriter, r *http.Request) {
		log.Printf("serveStatic: %s\n", r.URL.Path)
		if r.URL.Path == "/" {
			http.ServeFile(w, r, cfg.StaticDir + "/index.html")
			return
		}
		w.Header().Set("Cache-Control", cacheHeader)
		handler := handlers.Get(r.URL.Path)
		if handler == nil {
			http.Error(w, "not found", http.StatusNotFound)
		} else {
			handler(w, r)
		}
	}//-- end return
}//-- end func makeStaticServer

func (svr *DefaultServer) ServeStatic (w http.ResponseWriter,
		r *http.Request) {
	svr.staticServer(w, r)
}//-- end Webapp.serveStatic

