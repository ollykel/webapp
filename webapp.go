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
	"strings"
	"net/http"
	"context"
	"encoding/json"
	"encoding/xml"
	"./wapputils"
	"./model"
)

type Config struct {
	Port string
	Index string
	StaticDir string
	Database DatabaseConfig//-- see database.go
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
	db *database
}//-- end Webapp struct

func (app *Webapp) serveStatic(w http.ResponseWriter, r *http.Request) {
	log.Printf("serveStatic: %s\n", r.URL.Path)
	filename := r.URL.Path[1:]
	handler := wapputils.CacheFileServer(filename)
	app.mux.HandleFunc(r.URL.Path, handler)
	handler(w, r)
}//-- end Webapp.serveStatic

type Middleware func(http.ResponseWriter, *http.Request) bool

type MiddlewareBuilder func(*Webapp) Middleware

func (app *Webapp) AddMiddleware(builders ...MiddlewareBuilder) {
	additions := make([]Middleware, len(builders))
	for i := range builders {
		additions[i] = builders[i](app)
	}
	app.middleware = append(app.middleware, additions...)
}//-- end Webapp.AddMiddleware

func (app *Webapp) handleMiddleware(w http.ResponseWriter,
		r *http.Request) bool {
	shouldContinue := true
	for _, mware := range app.middleware {
		shouldContinue = mware(w, r)
		if !shouldContinue { return false }
	}//-- end for el
	return true
}//-- end Webapp.handleMiddleware

type AppHandler func(*Webapp) http.HandlerFunc

type Controller func(map[string]string) (int, string)
type View func(map[string][]string) (map[string]interface{}, int, string)

/*
type Methods struct {
	Get, Post, Put, Delete AppHandler
}//-- end Methods struct
*/

func (app *Webapp) HandleFunc(path string, handler http.HandlerFunc) {
	app.mux.HandleFunc(path, func(w http.ResponseWriter, r *http.Request) {
		if app.handleMiddleware(w, r) { handler(w, r) }
	})
}//-- end func Webapp.HandleFunc

type Methods struct {
	Get View
	Post, Put, Delete Controller
}//-- end Methods struct

func (app *Webapp) HandleView (vw View) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if !app.handleMiddleware(w, r) { return }
		w.Header().Set("Content-Type", "application/json")
		r.ParseForm()
		output, code, msg := vw(r.Form)
		w.WriteHeader(code)
		if code != http.StatusOK {
			fmt.Fprintf(w, `{"Error": "%s"}`, msg)
		} else {
			encoder := json.NewEncoder(w)
			encoder.Encode(output)
		}
	}//-- end return
}//-- end func HandleView

type controllerStatus struct {
	Success bool
	Error string
}//-- end controllerStatus struct

func (app *Webapp) HandleController (control Controller) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if !app.handleMiddleware(w, r) { return }
		w.Header().Set("Content-Type", "application/json")
		decoder := json.NewDecoder(r.Body)
		data := make(map[string]string)
		decoder.Decode(data)
		code, msg := control(data)
		w.WriteHeader(code)
		status := controllerStatus{}
		if code != http.StatusOK {
			status.Success, status.Error = false, msg
		} else {
			status.Success = true
		}
		encoder := json.NewEncoder(w)
		encoder.Encode(&status)
	}//-- end return
}//-- end func HandleController

func (app *Webapp) Register(path string, methods *Methods) {
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

func (app *Webapp) PrepareQuery (query string, md *ModelDefinition,
		readRow RowScanner) (SqlQuerier, error) {
	return app.db.prepareQuery(query, md, readRow)
}//-- end Webapp.PrepareQuery

func (app *Webapp) PrepareStmt (query string,
		md *ModelDefinition) (SqlStmt, error) {
	return app.db.prepareStmt(query, md)
}//-- end Webapp.PrepareStmt

func (app *Webapp) MakeQuery (query string,
		md *ModelDefinition) (SqlQuery, error) {
	return app.db.makeQuery(query, md)
}//-- end Webapp.MakeQuery

type Model interface {
	model.Model
	Init(*Webapp) error
}//-- end Model interface

func getModelFieldnames (fields map[string]string) []string {
	fieldNames := make([]string, len(fields))
	i := 0
	for key := range fields {
		fieldNames[i] = key
		i++
	}
	return fieldNames
}//-- end func getModelFieldnames

type ModelDefinition struct {
	Tablename func() string
	Fields func() map[string]string
	Constraints func() map[string]string
	Init func(*Webapp) error
}//-- end ModelDefinition struct

type ModelWrapper struct {
	def *ModelDefinition
}//-- end ModelWrapper struct

func (wrapper *ModelWrapper) Tablename () string {
	return wrapper.def.Tablename()
}//-- end ModelWrapper.Tablename

func (wrapper *ModelWrapper) Fields () map[string]string {
	return wrapper.def.Fields()
}

func (wrapper *ModelWrapper) Constraints () map[string]string {
	return wrapper.def.Constraints()
}

func (wrapper *ModelWrapper) Init (app *Webapp) error {
	return wrapper.def.Init(app)
}

func (app *Webapp) RegisterModels (mods []*ModelDefinition) error {
	var err error
	for _, mod := range mods {
		err = app.db.RegisterModel(&ModelWrapper{def: mod})
		if err != nil { return err }
		err = mod.Init(app)
		if err != nil { return err }
	}//-- end for range mods
	return nil
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
	app.db, err = initDatabase(config.Database.Driver,
		config.Database.Name, config.Database.User,
		config.Database.Password)
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

