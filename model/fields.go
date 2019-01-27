package model

import (
	"fmt"
	"errors"
	"strings"
)

type FieldType int

// FieldType enums
const (
	Integer FieldType = 1
	Int FieldType = 2
	TinyInt FieldType = 4
	SmallInt FieldType = 8
	BigInt FieldType = 16
	Char FieldType = 32
	Varchar FieldType = 64
	Text FieldType = 128
	Binary FieldType = 256
	Datetime FieldType = 512
	// Type Categories
	integerType FieldType = (Integer | Int | TinyInt | SmallInt | BigInt)
	lengthedType FieldType = (Char | Varchar)
)//-- end FieldType enums

func (ft FieldType) getName () (string, error) {
	switch (ft) {
		case 0:
			return "", errors.New("FieldType cannot be zero")
		case Integer, Int:
			return "INT", nil
		case TinyInt:
			return "TINYINT", nil
		case SmallInt:
			return "SMALLINT", nil
		case BigInt:
			return "BIGINT", nil
		case Char:
			return "CHAR", nil
		case Varchar:
			return "VARCHAR", nil
		case Text:
			return "TEXT", nil
		case Binary:
			return "BLOB", nil
		case Datetime:
			return "DATETIME", nil
		default:
			return "", fmt.Errorf("unrecognized FieldType (%d)", ft)
	}//-- end switch
}//-- end func getFieldName

func (ft FieldType) hasLength () bool { return ft & lengthedType != 0 }

type OnChangeBehavior int

const (
	Cascade OnChangeBehavior = 1
	SetNull OnChangeBehavior = 2
	Restrict OnChangeBehavior = 3
	NoAction OnChangeBehavior = 4
)//-- end OnChangeBehavior enums

func (beh OnChangeBehavior) getName () (string, error) {
	switch (beh) {
		case 0:
			return "", nil
		case Cascade:
			return "CASCADE", nil
		case SetNull:
			return "SET NULL", nil
		case Restrict:
			return "RESTRICT", nil
		case NoAction:
			return "NO ACTION", nil
		default:
			return "", fmt.Errorf("unrecognized behavior (%d)", beh)
	}//-- end switch
}//-- end func OnChangeBehavior.getName

type Field struct {
	Name string
	Type FieldType
	Length int
	Null, AutoIncrement, Unique bool
	Reference string
	OnDelete OnChangeBehavior
	OnUpdate OnChangeBehavior
}//-- end Field struct

func (fd *Field) buildFgnKey () (string, error) {
	output := []string{"FOREIGN KEY", "", "REFERENCES", fd.Reference,
		"(id)", "", ""}
	output[1] = fmt.Sprintf("(%s)", fd.Name)
	if fd.OnDelete != 0 {
		onDelete, err := fd.OnDelete.getName()
		if err != nil { return "", err }
		output[5] = fmt.Sprintf("ON DELETE %s", onDelete)
	}
	if fd.OnUpdate != 0 {
		onUpdate, err := fd.OnUpdate.getName()
		if err != nil { return "", err }
		output[6] = fmt.Sprintf("ON UPDATE %s", onUpdate)
	}
	return strings.Join(output, " "), nil
}//-- end Field.buildFgnKey

const defaultIdentity = "id BIGINT NOT NULL AUTO_INCREMENT PRIMARY KEY"

func (fd *Field) ToSchema () (string, error) {
	if fd.Reference != "" { fd.Type = BigInt }
	builder := strings.Builder{}
	builder.WriteString(fd.Name + " ")
	typeName, err := fd.Type.getName()
	if err != nil { return "", err }
	builder.WriteString(typeName + " ")
	if fd.Type.hasLength() { fmt.Fprintf(&builder, "(%d) ", fd.Length) }
	if !fd.Null { builder.WriteString("NOT NULL ") }
	if fd.AutoIncrement { builder.WriteString("AUTO_INCREMENT ") }
	if fd.Unique { builder.WriteString("UNIQUE KEY ") }
	return builder.String()[:builder.Len() - 1], nil
}//-- end Field.ToSchema

