package model

import (
	"os"
	"fmt"
	"log"
)

type Scannable interface {
	Scan(...interface{})
}//-- end Scannable interface

type Model interface {
	Schema () string
	Scan (Scannable)
}//-- end Model interface

