package model

import (
	"fmt"
	"strings"
)

type Scannable interface {
	Scan (...interface{}) error
}//-- end Scannable interface

/*
type Field struct {
	Name, Attribs string
}//-- end type Field
*/

type Model interface {
	Tablename () string
	Fields () []Field
}//-- end Model interface

type Sqlizable interface {
	Append (Scannable) error
	Clear () error
}//-- end Sqlizable interface

const fgnKeyFmt = "FOREIGN KEY (%s) REFERENCES %s (id)"

func Schema(mod Model) string {
	fields := mod.Fields()
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
	return fmt.Sprintf("CREATE TABLE %s (%s)", mod.Tablename(),
		strings.Join(schemas, ", "))
}//-- end func Schema

