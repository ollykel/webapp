package webapp

import (
	"database/sql"
	"strings"
	"container/list"
	"encoding/json"
)

type DatabaseConfig struct {
	Driver, Name, User, Password string
}

type Scannable interface {
	Scan(dest ...interface{}) error
}//-- end Scannable interface

type database struct {
	pool *sql.DB
	statements map[string]*sql.Stmt
}//-- end database struct

func initDatabase (driver, name, user, pass string) (*database, error) {
	dataSource := strings.Join([]string{name, user, pass}, "/")
	pool, err := sql.Open(driver, dataSource)
	if err != nil { return nil, err }
	return &database{pool: pool,
		statements: make(map[string]*sql.Stmt)}, nil
}//-- end func initDatabase

func (db *database) registerQuery (name, query string) error {
	statement, err := db.pool.Prepare(query)
	if err != nil { return err }
	db.statements[name] = statement
	return nil
}//-- end func database.registerQuery

type SqlQuerier func(...interface{}) (*list.List, error)

type RowScanner func(Scannable) interface{}

func (db *database) prepareQuery (query string,
		readRow RowScanner) (SqlQuerier, error) {
	stmt, err := db.pool.Prepare(query)
	if err != nil { return nil, err }
	return func(a ...interface{}) (*list.List, error) {
		rows, err := stmt.Query(a...)
		if err != nil { return nil, err }
		results := list.New()
		for rows.Next() {
			results.PushBack(readRow(rows))
		}//-- end for rows.Next
		return results, nil
	}, nil//-- end return
}//-- end database.prepareQuery

func (querier SqlQuerier) toJSON (a ...interface{}) (string, error) {
	data, err := querier(a...)
	if err != nil { return "", err }
	builder := strings.Builder{}
	encoder := json.NewEncoder(&builder)
	builder.WriteString("[")
	for el := data.Front(); el != nil; el = el.Next() {
		encoder.Encode(el.Value)
		builder.WriteString(",")
	}//-- end for el
	builder.WriteString("]")
	return builder.String(), nil
}//-- end func SqlQuerier.toJSON

