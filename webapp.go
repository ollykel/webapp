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
	"strings"
	"net/http"
	"context"
	"encoding/json"
	"encoding/xml"
	"./wapputils"
	"./database"
	"./model"
)

type Config struct {
	Port string
	Index string
	StaticDir string
	Database database.Config//-- see database.go
	Handlers map[string]http.HandlerFunc
}

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

type Webapp struct {
	server *http.Server//-- use *ServeMux as handler
	mux *http.ServeMux
	middleware []Middleware
	db *database.Database
}//-- end Webapp struct

func (app *Webapp) serveStatic(w http.ResponseWriter, r *http.Request) {
	log.Printf("serveStatic: %s\n", r.URL.Path)
	filename := r.URL.Path[1:]
	http.ServeFile(w, r, filename)
	/*
	handler := wapputils.CacheFileServer(filename)
	app.mux.HandleFunc(r.URL.Path, handler)
	handler(w, r)
	*/
}//-- end Webapp.serveStatic

type ReqData map[string]string

type Middleware func(http.ResponseWriter, *http.Request, ReqData) bool

func (app *Webapp) AddMiddleware(additions ...Middleware) {
	app.middleware = append(app.middleware, additions...)
}//-- end Webapp.AddMiddleware

func (app *Webapp) handleMiddleware(w http.ResponseWriter, r *http.Request,
		data ReqData) bool {
	for _, mware := range app.middleware {
		if !mware(w, r, data) { return false }
	}//-- end for mware
	return true
}//-- end Webapp.handleMiddleware

type AppHandler func(*Webapp) http.HandlerFunc

type Controller func(http.ResponseWriter, *http.Request, ReqData)
type View func(http.ResponseWriter, *http.Request, ReqData)

func (app *Webapp) HandleFunc(path string, handler http.HandlerFunc) {
	app.mux.HandleFunc(path, func(w http.ResponseWriter, r *http.Request) {
		if app.handleMiddleware(w, r, make(ReqData)) { handler(w, r) }
	})
}//-- end func Webapp.HandleFunc

type Methods struct {
	Handler http.HandlerFunc
	Get View
	Post, Put, Delete Controller
}//-- end Methods struct

func (app *Webapp) HandleView (vw View) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		data := make(ReqData)
		if app.handleMiddleware(w, r, data) { vw(w, r, data) }
	}//-- end return
}//-- end func HandleView

type controllerStatus struct {
	Success bool
	Error string
}//-- end controllerStatus struct

func (app *Webapp) HandleController (control Controller) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		data := make(ReqData)
		if app.handleMiddleware(w, r, data) { control(w, r, data) }
	}//-- end return
}//-- end func HandleController

func (app *Webapp) Register(path string, methods *Methods) {
	if methods.Handler != nil {
		app.HandleFunc(path, methods.Handler)
		return
	}
	handleDefault := func(w http.ResponseWriter, _ *http.Request) {
		http.Error(w, "bad request", http.StatusBadRequest)
	}//-- end handleDefault
	handleGet, handlePost, handlePut, handleDelete := handleDefault,
		handleDefault, handleDefault, handleDefault
	if methods.Get != nil { handleGet = app.HandleView(methods.Get) }
	if methods.Post != nil {
		handlePost = app.HandleController(methods.Post)
	}
	if methods.Put != nil { handlePut = app.HandleController(methods.Put) }
	if methods.Delete != nil {
		handleDelete = app.HandleController(methods.Delete)
	}
	app.mux.HandleFunc(path, func(w http.ResponseWriter, r *http.Request) {
		methodName := strings.ToUpper(r.Method)
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
}//-- end Webapp.Register

func (app *Webapp) RegisterMethods (methods map[string]*Methods) {
	for path, method := range methods {
		app.Register(path, method)
	}//-- end for range handlers
}//-- end func Webapp.RegisterMethods

func (app *Webapp) RegisterModels (mods []*model.Definition) (err error) {
	for _, mod := range mods {
		err = app.db.RegisterModel(mod)
		if err != nil { return }
		err = mod.Init(app.db)
		if err != nil { return }
	}//-- end for range mods
	return
}//-- end Webapp.RegisterModels

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
	app.db, err = database.New(&config.Database)
	if err != nil {
		return nil, err
	}
	app.middleware = make([]Middleware, 0)
	app.mux = http.NewServeMux()
	app.mux.HandleFunc("/", wapputils.CacheFileServer(config.Index))
	app.mux.HandleFunc(config.StaticDir, app.serveStatic)
	app.server = &http.Server{Addr: config.Port, Handler: app.mux}
	return app, nil
}//-- end func Init

