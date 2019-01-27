package model

import (
	"testing"
	"fmt"
)

type Foobar struct {
	Foo, Bar string
}//-- end Foobar struct

func (fb *Foobar) Tablename () string { return "foobar" }

func (fb *Foobar) Fields () []Field {
	return []Field {
		Field{Name: "name", Type: Varchar, Length: 64, Unique: true},
		Field{Name: "number", Type: Int, Null: true},
		Field{Name: "employer", Null: true, Reference: "employers",
			OnDelete: SetNull},
		Field{Name: "income", Type: Int}}
}//-- end Foobar.Fields

func (fb *Foobar) Scan(row Scannable) {
	row.Scan(nil, &fb.Foo, &fb.Bar)
}//-- end Foobar.Scan

func TestSchema (t *testing.T) {
	fmt.Print("Testing Schema()...\n")
	foobar := Foobar{Foo: "foo", Bar: "bar"}
	schema := Schema(&foobar)
	fmt.Printf("Schema: %s\n", schema)
	fmt.Print("Done testing Schema()\n\n")
}//-- end TestSchema

