package gromSqlite3

import (
	"errors"
	"fmt"
	"gorm.io/gorm"
)

// Table for store overridden days.
// By default all Mondays, Tuesdays, Wednesdays, Thursdays and Fridays are workdays
// and Saturdays and Sundays are not working days.
// Overridden days processed as opposite day type.
type WorkdayOverride struct {
	Day        int64 `gorm:"column:day;primaryKey"` // Day number since 1970.01.01 .
	Overridden bool  `gorm:"column:overridden;not null"`
}

// TableName overrides the table name to `workdayOverride` (for gorm).
func (WorkdayOverride) TableName() string {
	return "workdayOverride"
}

// Set new overridden day. If day already overridden do nothing.
func (db DB) SetWorkdayOverride(day int64) {
	// Check if day already overridden.
	if isWorkdayOverridden(db.Instance, day) {
		return
	}

	// Override day.
	workingDay := WorkdayOverride{
		Day:        day,
		Overridden: true,
	}
	db.Instance.Create(&workingDay)
}

// Remove overridden day. If day not overridden do nothing.
func (db DB) RemoveWorkdayOverride(day int64) {
	// Check if day overridden.
	if !isWorkdayOverridden(db.Instance, day) {
		return
	}

	// Remove override.
	db.Instance.Where("day = ?", day).Delete(WorkdayOverride{})
}

// Return list of all overridden days from specified range.
// If specified range not contain overridden days, return empty slice.
// initialDay must be >= 0 and sequenceLen mast be > 0.
func (db DB) GetOverrideByDaySequence(initialDay, sequenceLen int64) ([]int64, error) {
	// Check provided initialDay and sequenceLen.
	if initialDay < 0 {
		// TODO - create error variable
		return nil, errors.New(fmt.Sprintf("ivalid initial day '%v'", initialDay))
	}
	if sequenceLen < 1 {
		// TODO - create error variable
		return nil, errors.New(fmt.Sprintf("ivalid sequence len '%v'", sequenceLen))
	}

	overriddenDayList := make([]WorkdayOverride, 0, 16)
	db.Instance.Where("day >= ? and day < ?", initialDay, initialDay+sequenceLen).Find(&overriddenDayList)

	if len(overriddenDayList) == 0 {
		return make([]int64, 0, 0), nil
	}

	dayList := make([]int64, 0, 16)
	for _, overriddenDay := range overriddenDayList {
		dayList = append(dayList, overriddenDay.Day)
	}

	return dayList, nil
}

// Check if workday overridden.
func isWorkdayOverridden(db *gorm.DB, day int64) bool {
	workingDay := WorkdayOverride{
		Day:        0,
		Overridden: false,
	}
	db.Where("day = ?", day).First(&workingDay)
	return workingDay.Overridden
}
