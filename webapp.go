/**
 * @author Oliver Kelton, oakelton@gmail.com
 * @date Dec 22, 2018
 * Package to serve a single-page webapp, intended to be built
 * using a js framework such as React
 */

package webapp

import (
	"os"
	"fmt"
	"log"
	// "io"
	"io/ioutil"
	"strings"
	"container/list"
	"net/http"
	"context"
	"encoding/json"
	"encoding/xml"
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

type Config struct {
	Port string
	Index string
	StaticDir string
	Database DatabaseConfig//-- see database.go
	Handlers map[string]http.HandlerFunc
}//-- end Config struct

type decoder interface {
	Decode (interface{}) error
}//-- end Decoder interface

func LoadConfig (filename string) (*Config, error) {
	file, err := os.Open(filename)
	if err != nil { return nil, err }
	defer file.Close()
	path := strings.Split(filename, ".")
	ext := path[len(path) - 1]
	var dec decoder
	switch (ext) {
		case "json":
			dec = json.NewDecoder(file)
			break
		case "xml":
			dec = xml.NewDecoder(file)
			break
		default:
			return nil, fmt.Errorf(`Invalid file type "%s"`, ext)
	}//-- end switch
	config := new(Config)
	err = dec.Decode(config)
	if err != nil { return nil, err }
	return config, nil
}//-- end func LoadConfig

func setFileType(filename string) string {
	path := strings.Split(filename, ".")
	ext := path[len(path) - 1]
	fileType, exists := file_types[ext]
	log.Printf("File type: %s\n", fileType)
	if !exists { return file_types["default"] }
	return fileType
}//-- end func setFileType

func cacheFileServer (filename string) http.HandlerFunc {
	content, err := ioutil.ReadFile(filename)
	if err != nil { return http.NotFound }
	fileType := setFileType(filename)
	return func(w http.ResponseWriter, r *http.Request) {
		log.Printf("cached server: %s\n", r.URL.Path)
		w.Header().Set("Content-Type", fileType)
		w.Write(content)
	}//-- end return for existing file
}//-- end func cacheFileServer

func ServeJSON(w http.ResponseWriter, r *http.Request, item interface{}) {
	encoder := json.NewEncoder(w)
	err := encoder.Encode(item)
	if err != nil {
		http.Error(w, "internal server error",
			http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
}//-- end func ServeJSON

type Webapp struct {
	server *http.Server//-- use *ServeMux as handler
	mux *http.ServeMux
	middleware *list.List
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

type Middleware func(http.ResponseWriter, *http.Request) bool

type MiddlewareBuilder func(*Webapp) Middleware

func (app *Webapp) AddMiddleware(builder MiddlewareBuilder) {
	mware := builder(app)
	app.middleware.PushBack(mware)
}//-- end Webapp.AddMiddleware

func (app *Webapp) handleMiddleware(w http.ResponseWriter,
		r *http.Request) bool {
	shouldContinue := true
	for el := app.middleware.Front(); el != nil; el = el.Next() {
		shouldContinue = el.Value.(Middleware)(w, r)
		if !shouldContinue { return false }
	}//-- end for el
	return true
}//-- end Webapp.handleMiddleware

type AppHandler func(*Webapp) http.HandlerFunc

type Methods struct {
	Get, Post, Put, Delete AppHandler
}//-- end Methods struct

func (app *Webapp) Register(path string, methods *Methods) {
	handleDefault := func(w http.ResponseWriter, _ *http.Request) {
		http.Error(w, "bad request", http.StatusBadRequest)
	}//-- end handleDefault
	handleGet, handlePost, handlePut, handleDelete := handleDefault,
		handleDefault, handleDefault, handleDefault
	if methods.Get != nil { handleGet = methods.Get(app) }
	if methods.Post != nil { handlePost = methods.Post(app) }
	if methods.Put != nil { handlePut = methods.Put(app) }
	if methods.Delete != nil { handleDelete = methods.Delete(app) }
	app.mux.HandleFunc(path, func(w http.ResponseWriter, r *http.Request) {
		methodName := strings.ToUpper(r.Method)
		if !app.handleMiddleware(w, r) { return }
		switch (methodName) {
			case "GET":
				handleGet(w, r)
			case "POST":
				handlePost(w, r)
			case "PUT":
				handlePut(w, r)
			case "DELETE":
				handleDelete(w, r)
			default:
				handleDefault(w, r)
		}//-- end switch
	});//-- end HandleFunc
}//-- end Webapp.RegisterMethods

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
	app.middleware = list.New()
	app.mux = http.NewServeMux()
	app.mux.HandleFunc("/", cacheFileServer(config.Index))
	app.mux.HandleFunc(config.StaticDir, app.serveStatic)
	app.server = &http.Server{Addr: config.Port, Handler: app.mux}
	return app, nil
}//-- end func Init

