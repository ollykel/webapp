package webapp

import (
	"log"
	"database/sql"
	"context"
	"strings"
	"encoding/json"
	"encoding/xml"
	"./model"
)

type DatabaseConfig struct {
	Driver, DataSource, Name, User, Password string
}

type Scannable interface {
	Scan(dest ...interface{}) error
}//-- end Scannable interface

type database struct {
	pool *sql.DB
}//-- end database struct

func initDatabase (cfg *DatabaseConfig) (*database, error) {
	dataSource := cfg.DataSource
	pool, err := sql.Open(cfg.Driver, dataSource)
	if err != nil { return nil, err }
	return &database{pool: pool}, nil
}//-- end func initDatabase

type SqlQuerier func(...interface{}) ([]interface{}, error)

type SqlStmt func(model.Sqlizable, ...interface{}) error

type SqlQuery func(model.Sqlizable, ...interface{}) error

type RowScanner func(Scannable) interface{}

func parseQuery(query string, md *ModelDefinition) string {
	finalQuery := strings.Replace(query, "%TABLE%", md.Tablename(), -1)
	fieldNames := getModelFieldnames(md.Fields())
	finalQuery = strings.Replace(finalQuery, "%FIELDS%",
		strings.Join(fieldNames, ", "), -1)
	return finalQuery
}//-- end func parseQuery

func (db *database) prepareQuery (query string, md *ModelDefinition,
		readRow RowScanner) (SqlQuerier, error) {
	finalQuery := parseQuery(query, md)
	stmt, err := db.pool.Prepare(finalQuery)
	if err != nil { return nil, err }
	return func(a ...interface{}) ([]interface{}, error) {
		rows, err := stmt.Query(a...)
		if err != nil { return nil, err }
		results := make([]interface{}, 0)
		var nxt interface{}
		for rows.Next() {
			log.Print("foo!")
			nxt = readRow(rows)
			if nxt != nil { results = append(results, nxt) }
		}//-- end for rows.Next
		return results, nil
	}, nil//-- end return
}//-- end database.prepareQuery

func (db *database) prepareStmt (query string,
		md *ModelDefinition) (SqlStmt, error) {
	finalQuery := parseQuery(query, md)
	ctx := context.Background()
	stmt, err := db.pool.PrepareContext(ctx, finalQuery)
	if err != nil { return nil, err }
	var scanner func(model.Sqlizable, ...interface{}) error
	scanner = func(dest model.Sqlizable, a ...interface{}) error {
		cont := context.Background()
		rows, err := stmt.QueryContext(cont, a...)
		if err != nil { return err }
		defer rows.Close()
		if dest != nil {
			for rows.Next() {
				err = dest.Append(rows)
				if err != nil { return err }
			}//-- end for rows.Next
		}
		return rows.Err()
	}//-- end func scanner
	return scanner, nil//-- end return
}//-- end database.prepareStmt

func (db *database) makeQuery (query string,
		md *ModelDefinition) (SqlQuery, error) {
	finalQuery := parseQuery(query, md)
	scanner := func(dest model.Sqlizable, a ...interface{}) error {
		ctx := context.Background()
		rows, err := db.pool.QueryContext(ctx, finalQuery, a...)
		if err != nil { return err }
		defer rows.Close()
		if dest != nil {
			for rows.Next() {
				err = dest.Append(rows)
				if err != nil { return err }
			}//-- end for rows.Next
		}
		return rows.Err()
	}//-- end func scanner
	return scanner, nil
}//-- end func database.makeQuery

type SqlCmd func(...interface{}) (sql.Result, error)

func (db *database) makeCmd (query string,
		md *ModelDefinition) (SqlCmd, error) {
	finalCmd := parseQuery(query, md)
	return func(a ...interface{}) (sql.Result, error) {
		ctx := context.Background()
		return db.pool.ExecContext(ctx, finalCmd, a...)
	}, nil
}//-- end func database.makeCmd

func (querier SqlQuerier) toJSON (a ...interface{}) (string, error) {
	data, err := querier(a...)
	if err != nil { return "", err }
	builder := strings.Builder{}
	encoder := json.NewEncoder(&builder)
	encoder.Encode(data)
	return builder.String(), nil
}//-- end func SqlQuerier.toJSON

func (querier SqlQuerier) toXML (a ...interface{}) (string, error) {
	data, err := querier(a...)
	if err != nil { return "", err }
	builder := strings.Builder{}
	encoder := xml.NewEncoder(&builder)
	encoder.Encode(data)
	return builder.String(), nil
}//-- end SqlQuerier.toXML

func (db *database) TableExists (name string) bool {
	row := db.pool.QueryRow("SHOW TABLES LIKE ?", name)
	table := ""
	err := row.Scan(&table)
	return err == nil
}//-- end database.TableExists

func (db *database) RegisterModel (mod model.Model) error {
	if db.TableExists(mod.Tablename()) { return nil }
	schema := model.Schema(mod)
	_, err := db.pool.Exec(schema)
	if err != nil { return err }
	return nil
}//-- end func Database.RegisterModel

