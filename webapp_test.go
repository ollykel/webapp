package webapp

import (
	"testing"
	"log"
	"time"
	"context"
	_"github.com/ziutek/mymysql/godrv"//-- mysql driver
)

var (
	DEFAULT_CONFIG = Config{
		Port: ":9090",
		Index: "public/main.html",
		StaticDir: "/public/",
		Database: DatabaseConfig{
			Driver: "mymysql",
			Name: "test_db",
			User: "test_user",
			Password: "test_password",
		},
	}//-- end DEFAULT_CONFIG
)

func TestCompile(t *testing.T) {
	log.Print("Hello, World!")
}//-- end func TestCompile

type Foobar struct {
	Name string
	Number int
}

func TestDatabase(t *testing.T) {
	log.Print("Testing database functions...\n")
	config := &DEFAULT_CONFIG
	app, err := Init(config)
	if err != nil { t.Error(err) }
	query := "SELECT name, number FROM foobar WHERE id < ?"
	parseRows := func(rows Scannable) interface{} {
		foob := new(Foobar)
		rows.Scan(&foob.Name, &foob.Number)
		return foob
	}//-- end parseRows
	querier, err := app.PrepareQuery(query, parseRows)
	if err != nil { t.Error(err) }
	jsonData, err := querier.toJSON(5)
	if err != nil { t.Error(err) }
	log.Printf("Results: %s\n", jsonData)
	log.Print("Done testing database functions\n\n")
}//-- end TestDatabase

func TestSimpleServer(t *testing.T) {
	log.Print("Testing simple server...\n")
	config := &DEFAULT_CONFIG
	app, err := Init(config)
	if err != nil {
		t.Errorf("%s", err)
	}
	stopper := func() {
		for i := 20; i != 0; i-- {
			if i % 5 == 0 { log.Printf("Shutting down in %d secs...\n", i) }
			time.Sleep(1000 * time.Millisecond)
		}//-- end for i
		app.Shutdown(context.Background())
	}//-- end stopper
	go stopper()
	app.ListenAndServe()
	log.Print("Done testing simple server\n\n")
}//-- end func TestSimpleServer

