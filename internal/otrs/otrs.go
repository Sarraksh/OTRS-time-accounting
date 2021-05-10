package otrs

import "time"

type Provider interface {
	// Get today data from OTRS BD.
	GetTodayData() ([]DayStatisticRow, error)
	// Get accounted work time and overtime for specified day.
	GetCustomDayData(day time.Time) ([]DayStatisticRow, error)
	// Close DB connection.
	Stop() error
}

// Used for get today data.
type DayStatisticRow struct {
	LastName             string
	WorkTime             int
	OverTime             int
	NotClosedTicketCount int
	LockedTicketCount    int
	OpenTicketCount      int
}
