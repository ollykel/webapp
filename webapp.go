/**
 * @author Oliver Kelton, oakelton@gmail.com
 * @date Dec 22, 2018
 * Package to serve a single-page webapp, intended to be built
 * using a js framework such as React
 */

package webapp

import (
	"os"
	"errors"
	"fmt"
	"time"
	// "log"
	"container/list"
	"strings"
	"net/http"
)

type Config struct {
	Port string
	Index string
	StaticDir string
	Database struct {
		Driver, Name, User, Password string
	}
	Handlers map[string]http.HandlerFunc
}//-- end Config struct

func cacheFileServer (filename string) http.HandlerFunc {
	fcache, err := newFileCache(filename)
	if err != nil { return http.NotFound }
	modtime := time.Now()
	return func(w http.ResponseWriter, r *http.Request) {
		fcache.Seek(0, io.SeekStart)
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

func MakePostHandler (fn PostHandler) {
	return func(w http.ResponseWriter, r *http.Request) {
		path := strings.Split(r.URL.Path[1:], "/")
		fn(w, r, path)
	}//-- end return
}//-- end func MakePostHandler

type Scannable interface {
	Scan(dest ...interface{}) error
}//-- end Scannable interface

type database struct {
	pool *sql.DB
	statements map[string]*sql.Stmt
}//-- end database struct

func initDatabase (driver, name, user, pass string) (*database, error) {
	dataSource := strings.Join([]string{name, user, pass}, "/")
	return &database{pool: sql.Open(driver, dataSource),
		statements: make(map[string]*sql.Stmt)}
}//-- end func initDatabase

func (db *database) registerQuery (name, query string) error {
	db.statements[name], err := db.pool.Prepare(query)
	return err
}//-- end func database.registerQuery

func (db *database) prepareQuery (query string,
	readRow func(Scannable) interface{}) func(...interface{}) (*list.List,
	error) {
	stmt, err := db.pool.Prepare(query)
	return func(a ...interface{}) *list.List {
		if err != nil { return nil, err }
		rows, err := stmt.Query(a...)
		if err != nil { return nil, err }
		results := list.New()
		for rows.Next() {
			results.PushBack(readRow(rows))
		}//-- end for rows.Next
		return results, nil
	}//-- end return
}//-- end database.prepareQuery

type Webapp struct {
	server *http.Server//-- use *ServeMux as handler
	staticCache map[string]([]byte)
	db *database
}//-- end Webapp struct

func (app *Webapp) serveStatic(w http.ResponseWriter, r *http.Request) {
	filename := r.URL.path
	app.server.Handler.HandleFunc(filename, cacheFileServer(filename))
}//-- end Webapp.serveStatic

func Init (config *Config) (*Webapp, error) {
	app := new(Webapp)
	if err != nil {
		return nil, err
	}
	app.staticCache = make(map[string]([]byte))
	app.db, err = initDatabase(config.Database.Driver, config.Database.Name,
		config.Database.User, config.Database.Password)
	if err != nil {
		return nil, err
	}
	mux := http.NewServeMux()
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		http.ServeFile(w, r, config.Index)
	})
	mux.HandleFunc(config.StaticDir, app.serveStatic)
	app.server = &http.Server{Addr: config.Port, Handler: mux}
	return app, nil
}//-- end func Init

