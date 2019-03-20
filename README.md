# Webapp Framework
A simple framework for developing webapps in Golang.

## Design
The framework expects the project to adhere to a Model-View-Controller
pattern. A basic skeleton for a project can be found at [INSERT URL HERE].

### Models
Each model in a project should be confined to its own sub-package within
a larger "models" package. As specified in the "model" package, a model
must provide a Definition consisting of three components: a Tablename,
a slice of Fields (a struct defined in model/fields.go), and an
initialization function named "Init".

Init must should take a Database (interface defined in model/model.go)
and create the functions necessary to query the application's database.

A Database provides three functions to initialize queries:
- MakeQuery: provides a function to make unprepared queries to the db
- MakeCmd: provides a function to make unprepared non-queries
- PrepareQuery: provides a function that executes a prepared statement

Each model should hold these functions in unexported global vars but
utilize them in exported functions. Packages outside the models package
should never interface with the database directly, only through the
models packages.

