package httpServer

import (
	"sync"
	"time"
)

// Interface for HTTP server.
type Provider interface {
	ListenAndServe(port string)
}

// Help receive today data from main service.
type TodayStatistic struct {
	UpdateTime time.Time
	Data       []TodayStatisticRow
	mx         sync.Mutex
}

type TodayStatisticRow struct {
	LastName           string
	WorkShiftColor     string
	TimeAccounted      int
	TimeAccountedColor string
	AllTicketCount     int
	ClosedTicketCount  int
	OpenTicketCount    int
	LastInGroup        bool
}

// Help receive week data from main service.
type WeekStatistic struct {
	HeaderColor []string
	Data        []WeekStatisticRow
}

type WeekStatisticRow struct {
	User          UserCell
	TimeAccounted []TimeAccounted
}

type UserCell struct {
	LastName       string
	WorkShiftColor string
	LastInGroup    bool
}

type TimeAccounted struct {
	Time             int64
	Overtime         int64
	IsOverTimeExists bool
	Color            string
}

// Safe set data.
func (ts *TodayStatistic) Update(tsr []TodayStatisticRow) {
	ts.mx.Lock()
	defer ts.mx.Unlock()

	ts.Data = tsr
	ts.UpdateTime = time.Now()
}

// Safe get data.
func (ts *TodayStatistic) Get() ([]TodayStatisticRow, time.Time) {
	ts.mx.Lock()
	defer ts.mx.Unlock()

	return ts.Data, ts.UpdateTime
}
