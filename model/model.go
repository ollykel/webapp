package model

import (
	"strings"
)

type Scannable interface {
	Scan (...interface{}) error
}//-- end Scannable interface

type Model interface {
	Tablename () string
	Fields () map[string]string
	Constraints () map[string]string
}//-- end Model interface

type Sqlizable interface {
	Append (Scannable)
}//-- end Sqlizable interface

func Schema(mod Model) string {
	output := strings.Builder{}
	output.WriteString("CREATE TABLE " + mod.Tablename() + " (")
	fields := mod.Fields()
	for key, val := range fields {
		output.WriteString(key + " " + val)
		output.WriteString(", ")
	}//-- end for range mod.Fields
	constraints := mod.Constraints()
	i := len(constraints)
	for key, val := range constraints {
		output.WriteString(key + " " + val)
		if i != 1 { output.WriteString(", ") }
		i--
	}
	output.WriteString(");")
	return output.String()
}//-- end func Schema

