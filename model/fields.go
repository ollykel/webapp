package model

import (
	"fmt"
	"errors"
	"strings"
)

type fieldType int

// fieldType enums
const (
	Integer fieldType = 1
	Int fieldType = 1 << 1
	TinyInt fieldType = 1 << 2
	SmallInt fieldType = 1 << 3
	BigInt fieldType = 1 << 4
	Char fieldType = 1 << 5
	Varchar fieldType = 1 << 6
	Text fieldType = 1 << 7
	TinyText fieldType = 1 << 8
	MediumText fieldType = 1 << 9
	LongText fieldType = 1 << 10
	Blob fieldType = 1 << 11
	Binary fieldType = Blob
	TinyBlob fieldType = 1 << 12
	MediumBlob fieldType = 1 << 13
	LongBlob fieldType = 1 << 14
	Datetime fieldType = 1 << 15
	JSON fieldType = 1 << 16
	Decimal fieldType = 1 << 17
	Numeric fieldType = 1 << 18
	Float fieldType = 1 << 19
	Double fieldType = 1 << 20
	Bit fieldType = 1 << 21
	Date fieldType = 1 << 22
	Timestamp fieldType = 1 << 23
	Time fieldType = 1 << 24
	Year fieldType = 1 << 25
	Binary fieldType = 1 << 26
	VarBinary fieldType = 1 << 27
	Geometry fieldType = 1 << 28
	Point fieldType = 1 << 29
	LineString fieldType = 1 << 30
	Polygon fieldType = 1 << 31
	MultiPoint fieldType = 1 << 32
	MultiLineString fieldType = 1 << 33
	MultiPolygon fieldType = 1 << 34
	GeometryCollection fieldType = 1 << 35
	// Type Categories
	integerType fieldType = (Integer | Int | TinyInt | SmallInt | BigInt)
	lengthedType fieldType = (Char | Varchar)
)//-- end fieldType enums

func (ft fieldType) getName () (string, error) {
	switch (ft) {
		case 0:
			return "", errors.New("fieldType cannot be zero")
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
		case TinyText:
			return "TINYTEXT", nil
		case MediumText:
			return "MEDIUMTEXT", nil
		case LongText:
			return "LONGTEXT", nil
		case Blob:
			return "BLOB", nil
		case TinyBlob:
			return "TINYBLOB", nil
		case MediumBlob:
			return "MEDIUMBLOB", nil
		case LongBlob:
			return "LONGBLOB", nil
		case Datetime:
			return "DATETIME", nil
		case JSON:
			return "JSON", nil
		//-- TODO: complete all cases
		default:
			return "", fmt.Errorf("unrecognized fieldType (%d)", ft)
	}//-- end switch
}//-- end func getFieldName

func (ft fieldType) hasLength () bool { return ft & lengthedType != 0 }

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
	Type fieldType
	Length int
	Null, AutoIncrement, Unique bool
	Reference string
	OnDelete OnChangeBehavior
	OnUpdate OnChangeBehavior
}//-- end Field struct

func (self *Field) Equals (other *Field) bool {
	return (self.Name == other.Name && self.Type == other.Type &&
		self.Length == other.Length && self.Null == other.Null &&
		self.AutoIncrement == other.AutoIncrement &&
		self.Unique == other.Unique && self.Reference == other.Reference &&
		self.OnDelete == other.OnDelete && self.OnUpdate == other.OnUpdate)
}//-- end Equals

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

