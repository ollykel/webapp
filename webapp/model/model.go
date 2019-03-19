package model

import (
	"fmt"
	"strings"
	"database/sql"
)

type Scannable interface {
	Scan (...interface{}) error
}//-- end Scannable interface

type Sqlizable interface {
	Append (Scannable) error
}//-- end Sqlizable interface

type SqlStmt func(Sqlizable, ...interface{}) error
type SqlQuery func(Sqlizable, ...interface{}) error
type SqlCmd func(...interface{}) (sql.Result, error)

type Database interface {
	PrepareStmt (string, *Definition) (SqlStmt, error)
	MakeQuery (string, *Definition) (SqlQuery, error)
	MakeCmd (string, *Definition) (SqlCmd, error)
}//-- end db interface

type Definition struct {
	Tablename string
	Fields []Field
	Init func(Database) error
}//-- end Definition struct

const fgnKeyFmt = "FOREIGN KEY (%s) REFERENCES %s (id)"

func (def *Definition) FieldNames () []string {
	fieldNames := make([]string, len(def.Fields) + 1)
	fieldNames[0] = "id"
	for i := range def.Fields {
		fieldNames[i + 1] = def.Fields[i].Name
	}
	return fieldNames
}//-- end func getModelFieldnames

func (mod *Definition) Schema() string {
	fields := mod.Fields
	fieldsSchema := make([]string, len(fields) + 1)
	fieldsSchema[0] = defaultIdentity
	constraints := make([]string, len(fields))
	numConstraints := 0
	for i, field := range fields {
		fieldsSchema[i + 1], _ = field.ToSchema()
		if field.Reference != "" {
			constraints[numConstraints], _ = field.buildFgnKey()
			numConstraints++
		}
	}//-- end for range mod.Fields
	schemas := append(fieldsSchema, constraints[:numConstraints]...)
	return fmt.Sprintf("CREATE TABLE %s (%s)", mod.Tablename,
		strings.Join(schemas, ", "))
}//-- end func Schema

func (first *Definition) Equals (second *Definition) bool {
	if first.Tablename != second.Tablename { return false }
	firstFields, secondFields := first.Fields, second.Fields
	if len(firstFields) != len(secondFields) { return false }
	for i, fd := range firstFields {
		if !fd.Equals(&secondFields[i]) { return false }
	}//-- end for i
	return true
}//-- end func Equal

type Count int//-- used for single-column COUNT() queries

func (ct *Count) Append (row Scannable) error {
	return row.Scan(ct)
}//-- end Count.Append

