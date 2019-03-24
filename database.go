package webapp

/**
 * @author Oliver Kelton, oakelton@gmail.com
 * @date Mar 18, 2019
 * Specification for database to be used by webapp.
 * Initialization function should take a DatabaseConfig struct as an
 * argument and return an object satisfying the Database interface.
 * Database interface must, among many things, be able to
 * prepare statements and commands based on model definitions.
 */

import (
	"github.com/ollykel/webapp/model"
)

type DatabaseConfig struct {
	Protocol string
	Address string
	DatabaseName string
	Username string
	Password string
}//-- end DatabaseConfig struct

type Database interface {
	// Initializes db connection, throws error on failure
	Init (config *DatabaseConfig) error
	// see model/model.go
	model.Database
	// tests whether a table by a given name exists
	TableExists (name string) bool
	// modifies database as needed according to model definition
	RegisterModel (def *model.Definition) error
}//-- end Database interface

