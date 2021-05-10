package internalDB

// Declare set of methods for interaction with internal DB.
// Day numbers are counted since 1970.01.01 .
type Provider interface {

	// Set new overridden day. If day already overridden do nothing.
	SetWorkdayOverride(day int64)
	// Remove overridden day. If day not overridden do nothing.
	RemoveWorkdayOverride(day int64)
	// Return list of all overridden days from specified range.
	// If specified range not contain overridden days, return empty slice.
	// initialDay must be >= 0 and sequenceLen mast be > 0.
	GetOverrideByDaySequence(initialDay, sequenceLen int64) ([]int64, error)

	// Add accounted time for one user by one day.
	// If data already exists, don't overwrite it.
	AddAccountedTime(lastName string, day, workTime, overTime int64)
	// Add accounted time for one user by one day.
	AddOrUpdateAccountedTime(lastName string, day, workTime, overTime int64)
	// Get accounted data for for provided day (list of users and accounted time).
	GetAccountedTimeByDay(day int64) []AccountedTime
	// Get accounted data for for provided day and last name.
	GetAccountedTimeByDayAndLastname(day int64, lastName string) (int64, int64)
}

// Format for return accounted time.
type AccountedTime struct {
	WorkTime int64 // Main work time in minutes.
	OverTime int64 // Overtime work time in minuets.
}
