package mysql

/**
 * @author Oliver Kelton, oakelton@gmail.com
 * @date Mar 24, 2019
 * @dependency github.com/ziutek/mymysql/godrv
 * Database wrapper for mysql that satisfies webapp's Database interface.
 */

import (
	"database/sql"
	"fmt"
	"context"
	"strings"
	"log"
	// database driver
	_"github.com/ziutek/mymysql/godrv"
	// local imports
	app "gopkg.in/ollykel/webapp.v0"
	"gopkg.in/ollykel/webapp.v0/model"
)

const (
	driverName = "mymysql"
	dataSourceFmt = "%s:%s*%s/%s/%s"
	//-- Protocol:Address*DatabaseName/Username/Password
	//-- User:Password@Protocol(Address)/DatabaseName
)

type Scannable interface {
	Scan(dest ...interface{}) error
}//-- end Scannable interface

type Database struct {
	pool *sql.DB
}//-- end Database struct

func (db *Database) Init (cfg *app.DatabaseConfig) (err error) {
	dataSource := fmt.Sprintf(dataSourceFmt, cfg.Protocol, cfg.Address,
		cfg.DatabaseName, cfg.Username, cfg.Password)
	db.pool, err = sql.Open(driverName, dataSource)
	if err != nil { return }
	err = db.pool.Ping()
	return
}//-- end func initDatabase

func parseQuery(query string, md *model.Definition) string {
	tableName := md.Tablename
	finalQuery := strings.Replace(query, "%TABLE%", tableName, -1)
	fieldNames := md.FieldNames()
	for i, nm := range fieldNames {
		fieldNames[i] = strings.Join([]string{tableName, nm}, ".")
	}
	finalQuery = strings.Replace(finalQuery, "%FIELDS%",
		strings.Join(fieldNames, ", "), -1)
	return finalQuery
}//-- end func parseQuery

func (db *Database) PrepareStmt (query string,
		md *model.Definition) (model.SqlStmt, error) {
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
}//-- end Database.PrepareStmt

func (db *Database) MakeQuery (query string,
		md *model.Definition) (model.SqlQuery, error) {
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
}//-- end func Database.MakeQuery

func (db *Database) MakeCmd (query string,
		md *model.Definition) (model.SqlCmd, error) {
	finalCmd := parseQuery(query, md)
	return func(a ...interface{}) (sql.Result, error) {
		ctx := context.Background()
		return db.pool.ExecContext(ctx, finalCmd, a...)
	}, nil
}//-- end func Database.makeCmd

func (db *Database) TableExists (name string) bool {
	row := db.pool.QueryRow("SHOW TABLES LIKE ?", name)
	table := ""
	err := row.Scan(&table)
	if err != nil {
		log.Print(err.Error())
	}
	return err == nil && table != ""
}//-- end Database.TableExists

func (db *Database) SaveModel (mod *model.Definition) error {
	// if !modelTrackerInitialized { initModelTrackers(db) }
	if db.TableExists(mod.Tablename) { return nil }
	schema := mod.Schema()
	_, err := db.pool.Exec(schema)
	if err != nil {
		log.Print(err.Error())
		return err
	}
	return nil
}//-- end func Database.SaveModel

func (db *Database) RegisterModel (mod *model.Definition) error {
	if !modelTrackerInitialized { initModelTrackers(db) }
	db.Migrate(mod)
	return nil
}//-- end Database.RegisterModel

