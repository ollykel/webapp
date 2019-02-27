package database

import (
	"log"
	"fmt"
	"encoding/json"
	"../model"
)

/**
 * model for tracking migrations between model versions
 */

const (
	modelTrackerTable = "__model_trackers"
)//-- end const

type modelTracker struct {
	Id int
	Name, Fields string
}//-- end type modelTracker

type modelTrackerData []modelTracker

func (mt *modelTracker) Append (row model.Scannable) error {
	return row.Scan(&mt.Id, &mt.Name, &mt.Fields)
}//-- end modelTracker.Append

var (
	modelTrackerInitialized bool = false
	getModelTracker model.SqlQuery
	updateModelTracker model.SqlCmd
	createModelTracker model.SqlCmd
	modelTrackerExists model.SqlQuery
)//-- end var

func (mt *modelTracker) Create () (err error) {
	_, err = createModelTracker(&mt.Name, &mt.Fields)
	return
}//-- end modelTracker.Create

func (mt *modelTracker) Update () (err error) {
	_, err = updateModelTracker(mt.Fields, mt.Name)
	return
}//-- end modelTracker.Update

func (mt *modelTracker) Exists () bool {
	counter := model.Count(0)
	err := modelTrackerExists(&counter, mt.Name)
	return err == nil && counter != 0
}//-- end modelTracker.Exists

func (mt *modelTracker) Save () error {
	if mt.Exists() { return mt.Update() }
	return mt.Create()
}//-- end modelTracker.Save

func (mt *modelTracker) Fetch () error {
	err := getModelTracker(mt, mt.Name)
	if err != nil { return err }
	if mt.Id == 0 { return fmt.Errorf("model %s not found", mt.Name) }
	return nil
}//-- end modelTracker.Fetch

func (mt *modelTracker) ToModelDefinition () *model.Definition {
	output := model.Definition{Tablename: mt.Name}
	err := json.Unmarshal([]byte(mt.Fields), &output.Fields)
	if err != nil { log.Print(err.Error()) }
	return &output
}//-- end func modelTracker.ToModelDefinition

func (mt *modelTracker) FromModelDefinition (def *model.Definition) {
	mt.Name = def.Tablename
	fields, err := json.Marshal(&def.Fields)
	if err != nil { log.Print(err.Error) }
	mt.Fields = string(fields)
}//-- end func modelTracker.FromModelDefinition

func initModelTrackers (db model.Database) (err error) {
	getModelTracker, err = db.MakeQuery(`SELECT %FIELDS% FROM %TABLE%
		WHERE name = ? LIMIT 1`, defineModelTracker())
	if err != nil { return }
	log.Print("Initialized getModelTracker...")
	updateModelTracker, err = db.MakeCmd(`UPDATE %TABLE% SET
		fields = ? WHERE name = ? LIMIT 1`, defineModelTracker())
	if err != nil { return }
	createModelTracker, err = db.MakeCmd(`INSERT INTO %TABLE%
		(name, fields) VALUES ( ? , ? )`, defineModelTracker())
	if err != nil { return }
	modelTrackerExists, err = db.MakeQuery(`SELECT COUNT(id) FROM
		%TABLE% WHERE name = ?`, defineModelTracker())
	if err != nil { return }
	modelTrackerInitialized = true
	return
}//-- end func initModelTrackers

func defineModelTracker () *model.Definition {
	return &model.Definition{
		Tablename: modelTrackerTable,
		Fields: []model.Field{
			model.Field{Name: "name", Type: model.Varchar, Length: 64,
				Unique: true},
			model.Field{Name: "fields", Type: model.Blob}},
		Init: initModelTrackers}//-- end return
}//-- end func defineModelTracker

type migrationType int

const (
	addMigration migrationType = 1
	modMigration migrationType = 2
)

type migration struct {
	Type migrationType
	Field *model.Field
}//-- end migration struct

func (mig *migration) Schema () string {
	var verb string
	switch (mig.Type) {
		case addMigration:
			verb = "ADD"
		case modMigration:
			verb = "MODIFY"
		default:
			log.Fatal(fmt.Sprintf("unrecognized migration (%d)", mig.Type))
	}//-- end switch
	fieldSchema, _ := mig.Field.ToSchema()
	return fmt.Sprintf("%s COLUMN %s", verb, fieldSchema)
}//-- end migration.Schema

func (db *Database) getMigrations (def *model.Definition) []migration {
	tracker := modelTracker{Name: def.Tablename}
	tracker.Fetch()
	if tracker.Id == 0 { return nil }
	origDef := tracker.ToModelDefinition()
	origFields := make(map[string]*model.Field)
	for i, fd := range origDef.Fields {
		origFields[fd.Name] = &origDef.Fields[i]
	}//-- end for range origDef.Fields
	migrations, numMigrations := make([]migration, len(def.Fields)), 0
	var mig *migration
	for i, fd := range def.Fields {
		mig = &migrations[numMigrations]
		if origFields[fd.Name] == nil {
			mig.Type, mig.Field = addMigration, &def.Fields[i]
			numMigrations++
		} else if !fd.Equals(origFields[fd.Name]) {
			mig.Type, mig.Field = modMigration, &def.Fields[i]
			numMigrations++
		}
	}//-- end for range def.Fields
	return migrations[:numMigrations]
}//-- end func Database.getMigrations

func (db *Database) Migrate (def *model.Definition) {
	var err error
	if !modelTrackerInitialized && !db.TableExists(modelTrackerTable) {
		err = db.SaveModel(defineModelTracker())
		if err != nil { log.Fatal(err) }
	}
	if !db.TableExists(def.Tablename) {
		db.SaveModel(def)
	} else {
		migrations := db.getMigrations(def)
		for _, mig := range migrations {
			log.Printf("Migration: %s", mig.Schema())
			_, err = db.pool.Exec(fmt.Sprintf("ALTER TABLE %s %s",
				def.Tablename, mig.Schema()))
			if err != nil { log.Fatal(err) }
		}//-- end for range migrations
	}
	tracker := modelTracker{}
	tracker.FromModelDefinition(def)
	tracker.Save()
}//-- end Database.diffModel

