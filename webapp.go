/**
 * @author Oliver Kelton, oakelton@gmail.com
 * @date Dec 22, 2018
 * Package to serve a single-page webapp, intended to be built
 * using a js framework such as React
 */

package webapp

import (
	"log"
	"io"
	"strings"
	"net/http"
	"context"
	"encoding/json"
)

var (
	file_types = map[string]string{
		"default": "text/plain",
		"txt": "text/plain",
		"html": "text/html",
		"css": "text/css",
		"js": "application/javascript",
		"csv": "text/csv",
		"gif": "image/gif",
		"ico": "image/x-icon",
		"jpeg": "image/jpeg",
		"jpg": "image/jpeg",
		"json": "application/json",
		"mpeg": "video/mpeg",
		"png": "image/png",
		"pdf": "application/pdf",
		"svg": "image/svg+xml",
		"tar": "application/x-tar",
		"tif": "image/tiff",
		"tiff": "image/tiff",
		"wav": "audio/wav",
		"xhtml": "application/xhtml+xml",
		"xml": "application/xml",
		"zip": "application/zip"}//-- end file_types
)

type DatabaseConfig struct {
	Driver, Name, User, Password string
}

type Config struct {
	Port string
	Index string
	StaticDir string
	Database DatabaseConfig
	Handlers map[string]http.HandlerFunc
}//-- end Config struct

func setFileType(filename string) string {
	path := strings.Split(filename, ".")
	ext := path[len(path) - 1]
	fileType, exists := file_types[ext]
	log.Printf("File type: %s\n", fileType)
	if !exists { return file_types["default"] }
	return fileType
}//-- end func setFileType

func cacheFileServer (filename string) http.HandlerFunc {
	fcache, err := newFileCache(filename)
	if err != nil { return http.NotFound }
	stat, _ := fcache.Stat()
	modtime := stat.ModTime()
	fileType := setFileType(filename)
	return func(w http.ResponseWriter, r *http.Request) {
		log.Printf("cached server: %s\n", r.URL.Path)
		fcache.Seek(0, io.SeekStart)
		w.Header().Set("Content-Type", fileType)
		http.ServeContent(w, r, filename, modtime, fcache)
	}//-- end return for existing file
}//-- end func cacheFileServer

type GetHandler func(http.ResponseWriter, *http.Request, []string)

type PostHandler func(http.ResponseWriter, *http.Request, []string)

func MakeGetHandler (fn GetHandler) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		path := strings.Split(r.URL.Path[1:], "/")
		fn(w, r, path)
	}//-- end return
}//-- end func MakeGetHandler

func MakePostHandler (fn PostHandler) func(http.ResponseWriter,
		*http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		path := strings.Split(r.URL.Path[1:], "/")
		fn(w, r, path)
	}//-- end return
}//-- end func MakePostHandler

func ServeJSON(w http.ResponseWriter, r *http.Request, item interface{}) {
	encoder := json.NewEncoder(w)
	err := encoder.Encode(item)
	if err != nil {
		http.Error(w, r, http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
}//-- end func ServeJSON

type Webapp struct {
	server *http.Server//-- use *ServeMux as handler
	mux *http.ServeMux
	staticCache map[string]([]byte)
	db *database
}//-- end Webapp struct

func (app *Webapp) serveStatic(w http.ResponseWriter, r *http.Request) {
	log.Printf("serveStatic: %s\n", r.URL.Path)
	filename := r.URL.Path[1:]
	handler := cacheFileServer(filename)
	app.mux.HandleFunc(r.URL.Path, handler)
	handler(w, r)
}//-- end Webapp.serveStatic

func (app *Webapp) HandleFunc(path string, handler http.HandlerFunc) {
	app.mux.HandleFunc(path, handler)
}//-- end func Webapp.HandleFunc

type AppHandler func(*Webapp) http.HandlerFunc

func (app *Webapp) Register(path string, handler AppHandler) {
	app.mux.HandleFunc(path, handler(app))
}//-- end Webapp.Register

func (app *Webapp) PrepareQuery (query string,
		readRow RowScanner) (SqlQuerier, error) {
	return app.db.prepareQuery(query, readRow)
}//-- end Webapp.PrepareQuery

func (app *Webapp) ListenAndServe() error {
	log.Printf("Server listening at %s...\n", app.server.Addr)
	return app.server.ListenAndServe()
}//-- end func Webapp.ListenAndServe

func (app *Webapp) Shutdown(ctx context.Context) error {
	return app.server.Shutdown(ctx)
}//-- end func Webapp.Shutdown

func Init (config *Config) (*Webapp, error) {
	var err error
	app := new(Webapp)
	app.staticCache = make(map[string]([]byte))
	app.db, err = initDatabase(config.Database.Driver,
		config.Database.Name, config.Database.User,
		config.Database.Password)
	if err != nil {
		return nil, err
	}
	app.mux = http.NewServeMux()
	app.mux.HandleFunc("/", cacheFileServer(config.Index))
	app.mux.HandleFunc(config.StaticDir, app.serveStatic)
	app.server = &http.Server{Addr: config.Port, Handler: app.mux}
	return app, nil
}//-- end func Init

