package gromSqlite3

import (
	"github.com/Sarraksh/OTRS-time-accounting/internal/internalDB"
	"gorm.io/gorm"
)

// Table for store accounted time.
type AccountedTime struct {
	Day      int64  `gorm:"column:day;primaryKey"`      // Day number since 1970.01.01 .
	LastName string `gorm:"column:lastName;primaryKey"` // User name.
	WorkTime int64  `gorm:"column:workTime;not null"`   // Main work time in minutes.
	OverTime int64  `gorm:"column:overTime;not null"`   // Overtime work time in minuets.
}

// TableName overrides the table name to `accountedTime` (for gorm).
func (AccountedTime) TableName() string {
	return "accountedTime"
}

// Add accounted time for one user by one day.
// If data already exists, don't overwrite it.
func (db DB) AddAccountedTime(lastName string, day, workTime, overTime int64) {
	// Check if time already accounted.
	// Do not add new data or rewrite old data if time already accounted.
	if isTimeAccounted(db.Instance, lastName, day) {
		// TODO - add custom error
		return
	}
	// Add new row.
	at := AccountedTime{
		Day:      day,
		LastName: lastName,
		WorkTime: workTime,
		OverTime: overTime,
	}
	db.Instance.Model(&AccountedTime{}).Create(&at)
}

// Add accounted time for one user by one day.
// If data already exists, overwrite it.
func (db DB) AddOrUpdateAccountedTime(lastName string, day, workTime, overTime int64) {
	at := AccountedTime{
		Day:      day,
		LastName: lastName,
		WorkTime: workTime,
		OverTime: overTime,
	}
	// Check if time already accounted.
	// Update old data if time already accounted.
	if isTimeAccounted(db.Instance, lastName, day) {
		db.Instance.Model(&AccountedTime{}).Where("day = ? and lastName = ?", day, lastName).Updates(&at)
		return
	}
	// Add new row.
	db.Instance.Model(&AccountedTime{}).Create(&at)
}

// Get accounted data for for provided day (list of users and accounted time).
func (db DB) GetAccountedTimeByDay(day int64) []internalDB.AccountedTime {
	atList := make([]internalDB.AccountedTime, 0, 32)
	db.Instance.Model(&AccountedTime{}).Where("day = ?", day).Find(&atList)
	// TODO - add custom error if list is empty slice
	return atList
}

// Get accounted data for for provided day and last name.
func (db DB) GetAccountedTimeByDayAndLastname(day int64, lastName string) (int64, int64) {
	at := AccountedTime{
		WorkTime: 0,
		OverTime: 0,
	}
	db.Instance.Model(&AccountedTime{}).Where("day = ? and lastName = ?", day, lastName).Find(&at)
	return at.WorkTime, at.OverTime
}

// Check if time already accounted.
func isTimeAccounted(db *gorm.DB, lastName string, day int64) bool {
	at := AccountedTime{Day: 0}
	db.Model(&AccountedTime{}).Where("day = ? and lastName = ?", day, lastName).First(&at)
	if at.Day != 0 {
		return true
	}
	return false
}
