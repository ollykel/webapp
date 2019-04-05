/**
 * @author Oliver Kelton, oakelton@gmail.com
 * @date Dec 22, 2018
 * Package to serve a single-page webapp, intended to be built
 * using a js framework such as React
 */

package webapp

import (
	"log"
	"strings"
	"net/http"
	"context"
	"time"
	"gopkg.in/ollykel/webapp.v0/wapputils"
	"gopkg.in/ollykel/webapp.v0/model"
)

/** see config.go for related funcs
type Config struct {
	Index string
	StaticDir string
	WaitSecs int
	Server ServerConfig//-- see server.go
	Database DatabaseConfig//-- see database.go
}
*/

type Handler interface {
	http.Handler
	HandleFunc (path string, handler func(w http.ResponseWriter,
		r *http.Request))
}//-- end Handler interface

type Webapp struct {
	server Server
	handler Handler//-- will go into server
	middleware []Middleware
	db Database
}//-- end Webapp struct

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
	app.handler.HandleFunc(path, func(w http.ResponseWriter, r *http.Request) {
		if app.handleMiddleware(w, r, make(ReqData)) { handler(w, r) }
	})
}//-- end func Webapp.HandleFunc

type Methods struct {
	Handler http.HandlerFunc
	Get View
	Post, Put, Patch, Delete Controller
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
	handleGet, handlePost := handleDefault, handleDefault
	handlePut, handlePatch := handleDefault, handleDefault
	handleDelete := handleDefault
	if methods.Get != nil { handleGet = app.HandleView(methods.Get) }
	if methods.Post != nil {
		handlePost = app.HandleController(methods.Post)
	}
	if methods.Put != nil { handlePut = app.HandleController(methods.Put) }
	if methods.Patch != nil {
		handlePatch = app.HandleController(methods.Patch)
	}
	if methods.Delete != nil {
		handleDelete = app.HandleController(methods.Delete)
	}
	app.handler.HandleFunc(path, func(w http.ResponseWriter, r *http.Request) {
		methodName := strings.ToUpper(r.Method)
		switch (methodName) {
			case "GET":
				handleGet(w, r)
			case "POST":
				handlePost(w, r)
			case "PUT":
				handlePut(w, r)
			case "PATCH":
				handlePatch(w, r)
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
	log.Printf("Server listening at %s...\n", app.server.GetAddr())
	return app.server.Serve()
}//-- end func Webapp.ListenAndServe

func (app *Webapp) Shutdown(ctx context.Context) error {
	return app.server.Shutdown(ctx)
}//-- end func Webapp.Shutdown

func Init (config *Config, svr Server, handler Handler,
		db Database) (app *Webapp, err error) {
	if config.WaitSecs > 0 {
		log.Printf("Waiting %d seconds...", config.WaitSecs)
		time.Sleep(time.Duration(config.WaitSecs) * time.Second)
	}
	app = new(Webapp)
	app.db = db
	err = db.Init(&config.Database)
	if err != nil { return nil, err }
	log.Print("Database reached successfully")
	app.middleware = make([]Middleware, 0)
	app.handler = handler
	err = svr.Init(&config.Server, app.handler)
	if err != nil { return nil, err }
	log.Print("Server initialized successfully")
	app.handler.HandleFunc("/", svr.ServeStatic)
	app.server = svr
	return app, nil
}//-- end func Init

