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
	"./resp"
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

type ReqData map[string]string

type Middleware func(*http.Request, ReqData) resp.Response

func (app *Webapp) AddMiddleware(additions ...Middleware) {
	app.middleware = append(app.middleware, additions...)
}//-- end Webapp.AddMiddleware

func (app *Webapp) handleMiddleware(r *http.Request,
		data ReqData) (res resp.Response) {
	for _, mware := range app.middleware {
		res = mware(r, data)
		if res != nil { return }
	}//-- end for mware
	return
}//-- end Webapp.handleMiddleware

type AppHandler func(*Webapp) http.HandlerFunc

type Controller func(ReqData) resp.Response
type View func(ReqData) resp.Response

/*
type Methods struct {
	Get, Post, Put, Delete AppHandler
}//-- end Methods struct
*/

func (app *Webapp) HandleFunc(path string, handler http.HandlerFunc) {
	app.mux.HandleFunc(path, func(w http.ResponseWriter, r *http.Request) {
		res := app.handleMiddleware(r, make(ReqData))
		if res != nil {
			res.Write(w)
			return
		}
		handler(w, r)
	})
}//-- end func Webapp.HandleFunc

type Methods struct {
	Handler http.HandlerFunc
	Get View
	Post, Put, Delete Controller
}//-- end Methods struct

func formToReqData (form map[string][]string) ReqData {
	output := make(ReqData)
	for key, val := range form {
		output[key] = strings.Join(val, " ")
	}//-- end for range form 
	return output
}//-- end func formToReqData

func (app *Webapp) HandleView (vw View) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		r.ParseForm()
		data := formToReqData(r.Form)
		res := app.handleMiddleware(r, data)
		if res != nil {
			res.Write(w)
			return
		}
		res = vw(data)
		if res != nil { res.Write(w) }
	}//-- end return
}//-- end func HandleView

type controllerStatus struct {
	Success bool
	Error string
}//-- end controllerStatus struct

func (app *Webapp) HandleController (control Controller) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		data := make(ReqData)
		decoder := json.NewDecoder(r.Body)
		decoder.Decode(data)
		response := app.handleMiddleware(r, data)
		if response != nil {
			response.Write(w)
			return
		}
		response = control(data)
		if response != nil { response.Write(w) }
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

func (app *Webapp) MakeCmd (query string,
		md *ModelDefinition) (SqlCmd, error) {
	return app.db.makeCmd(query, md)
}//-- end Webapp.MakeCmd

type Model interface {
	model.Model
	Init(*Webapp) error
}//-- end Model interface

func getModelFieldnames (fields []model.Field) []string {
	fieldNames := make([]string, len(fields) + 1)
	fieldNames[0] = "id"
	for i := range fields {
		fieldNames[i + 1] = fields[i].Name
	}
	return fieldNames
}//-- end func getModelFieldnames

type ModelDefinition struct {
	Tablename func() string
	Fields func() []model.Field
	Constraints func() map[string]string
	Init func(*Webapp) error
}//-- end ModelDefinition struct

type ModelWrapper struct {
	def *ModelDefinition
}//-- end ModelWrapper struct

func (wrapper *ModelWrapper) Tablename () string {
	return wrapper.def.Tablename()
}//-- end ModelWrapper.Tablename

func (wrapper *ModelWrapper) Fields () []model.Field {
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
	app.db, err = initDatabase(&config.Database)
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

