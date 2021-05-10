package gromSqlite3

import (
	"github.com/Sarraksh/OTRS-time-accounting/internal/internalDB"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

type DB struct {
	Instance *gorm.DB
}

func NewDB(fileName string) (internalDB.Provider, error) {
	// use default file name if not present.
	if fileName == "" {
		fileName = "sqlite.db"
	}

	// Initialise DB file (open or create).
	db, err := openDB(fileName)
	if err != nil {
		return nil, err
	}

	// Initialise DB schema.
	db, err = initialiseDBSchema(db)
	if err != nil {
		return nil, err
	}

	DBProvider := DB{Instance: db}

	return DBProvider, nil
}

func openDB(fileName string) (*gorm.DB, error) {
	db, err := gorm.Open(sqlite.Open(fileName), &gorm.Config{})
	if err != nil {
		return nil, err
	}
	return db, nil
}

func initialiseDBSchema(db *gorm.DB) (*gorm.DB, error) {
	//err := db.AutoMigrate(&DayTimeAccounted{})
	//if err != nil {
	//	return nil, err
	//}

	err := db.AutoMigrate(&WorkdayOverride{})
	if err != nil {
		return nil, err
	}

	err = db.AutoMigrate(&AccountedTime{})
	if err != nil {
		return nil, err
	}

	return db, nil
}
