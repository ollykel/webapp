package webapp

import (
	"log"
	"database/sql"
	"strings"
	"encoding/json"
	"encoding/xml"
	"./model"
)

type DatabaseConfig struct {
	Driver, Name, User, Password string
}

type Scannable interface {
	Scan(dest ...interface{}) error
}//-- end Scannable interface

type database struct {
	pool *sql.DB
}//-- end database struct

func initDatabase (driver, name, user, pass string) (*database, error) {
	dataSource := strings.Join([]string{name, user, pass}, "/")
	pool, err := sql.Open(driver, dataSource)
	if err != nil { return nil, err }
	return &database{pool: pool}, nil
}//-- end func initDatabase

type SqlQuerier func(...interface{}) ([]interface{}, error)

type SqlStmt func(model.Sqlizable, ...interface{}) error

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
	stmt, err := db.pool.Prepare(finalQuery)
	if err != nil { return nil, err }
	return func(dest model.Sqlizable, a ...interface{}) error {
		rows, err := stmt.Query(a...)
		if err != nil { return err }
		if dest != nil {
			for rows.Next() { dest.Append(rows) }//-- end for rows.Next
		}
		return nil
	}, nil//-- end return
}//-- end database.prepareStmt

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

