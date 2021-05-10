package service

import (
	"github.com/Sarraksh/OTRS-time-accounting/internal/httpServer"
	"github.com/Sarraksh/OTRS-time-accounting/internal/internalDB"
	"time"
)

// Uses in week calculation and table visualisation.
type Workday struct {
	Number    int64 // Day number since 1970.01.01 .
	IsWorkday bool  // True if workday false if day off.
}

// Uses in table visualisation.
type User struct {
	LastName    string
	WorkShift   string
	LastInGroup bool
}

// Calculate workweek from day list and overridden day list.
func CalculateWorkWeek(dayList, overriddenDayList []int64) []Workday {
	wd := make([]Workday, 0, 7) // Initial empty slice for week day list.
	var isOverridden bool       // Indicate if current day overridden.
	for i, day := range dayList {
		isOverridden = false                                   // Reset indicator for new day
		wd = append(wd, Workday{Number: day, IsWorkday: true}) // Add new day as workday.
		for _, overriddenDay := range overriddenDayList {
			if day == overriddenDay {
				isOverridden = true
			}
		}

		// Change day if it day off. By default 1-5 day is workdays, 6 and 7 is days off.
		// If 1-5 day overridden, it is considered as day off.
		// If 6 or 7 day overridden, it is considered as working day.
		if i < 5 && isOverridden == true || i >= 5 && isOverridden == false {
			wd[i].IsWorkday = false
		}
	}
	return wd
}

// Return function for usage in HTTP server.
func (s *Service) WeekConnector(weekOffset int64) func() (httpServer.WeekStatistic, error) {
	return func() (httpServer.WeekStatistic, error) {
		WeekDayList := getWeekDayList(weekOffset)
		OverriddenDayList, err := s.DB.GetOverrideByDaySequence(WeekDayList[0], 7)
		if err != nil {
			return httpServer.WeekStatistic{}, err
		}

		dayList := CalculateWorkWeek(WeekDayList, OverriddenDayList)

		var ws httpServer.WeekStatistic
		userOrder := getUserOrder()
		for _, user := range userOrder {
			ws.Data = append(ws.Data, httpServer.WeekStatisticRow{User: user, TimeAccounted: make([]httpServer.TimeAccounted, 8)})
		}

		ws = collectWeekData(s.DB, dayList, ws)

		return ws, nil
	}
}

// Collect data and assemble in correct order for show on web page.
func collectWeekData(db internalDB.Provider, dayList []Workday, ws httpServer.WeekStatistic) httpServer.WeekStatistic {
	var workTime, overTime int64
	for rowIndex, row := range ws.Data {
		for columnIndex, day := range dayList {
			// Get time data from internal DB and store into web data struct.
			workTime, overTime = db.GetAccountedTimeByDayAndLastname(day.Number, row.User.LastName)
			ws.Data[rowIndex].TimeAccounted[columnIndex+1].Time = ws.Data[rowIndex].TimeAccounted[columnIndex+1].Time + workTime
			ws.Data[rowIndex].TimeAccounted[columnIndex+1].Overtime = ws.Data[rowIndex].TimeAccounted[columnIndex+1].Overtime + overTime
			ws.Data[rowIndex].TimeAccounted[0].Time = ws.Data[rowIndex].TimeAccounted[0].Time + workTime + overTime

			// Define color for cell with accounted time for current day.
			switch {
			case !day.IsWorkday:
				ws.Data[rowIndex].TimeAccounted[columnIndex+1].Color = "good-grid-col"
			case ws.Data[rowIndex].TimeAccounted[columnIndex+1].Time+ws.Data[rowIndex].TimeAccounted[columnIndex+1].Overtime < 240:
				ws.Data[rowIndex].TimeAccounted[columnIndex+1].Color = "bad-grid-col"
			case ws.Data[rowIndex].TimeAccounted[columnIndex+1].Time+ws.Data[rowIndex].TimeAccounted[columnIndex+1].Overtime < 300:
				ws.Data[rowIndex].TimeAccounted[columnIndex+1].Color = "average-grid-col"
			default:
				ws.Data[rowIndex].TimeAccounted[columnIndex+1].Color = "good-grid-col"
			}

			// Controls the overtime visibility. Don't show if zero.
			if ws.Data[rowIndex].TimeAccounted[columnIndex+1].Overtime > 0 {
				ws.Data[rowIndex].TimeAccounted[columnIndex+1].IsOverTimeExists = true
			}
		}
	}

	// Calculate workday count and define colors for table title.
	var workdayCount int64 = 0
	ws.HeaderColor = make([]string, 8, 8)
	ws.HeaderColor[0] = "themed-grid-col"
	for i, day := range dayList {
		if day.IsWorkday {
			ws.HeaderColor[i+1] = "work-day-grid-col"
			workdayCount++
		} else {
			ws.HeaderColor[i+1] = "day-off-grid-col"
		}
	}

	// Define color for cell with accounted time for week.
	for rowIndex, _ := range ws.Data {
		switch {
		case ws.Data[rowIndex].TimeAccounted[0].Time < (240 * workdayCount):
			ws.Data[rowIndex].TimeAccounted[0].Color = "bad-grid-col"
		case ws.Data[rowIndex].TimeAccounted[0].Time < (300 * workdayCount):
			ws.Data[rowIndex].TimeAccounted[0].Color = "average-grid-col"
		default:
			ws.Data[rowIndex].TimeAccounted[0].Color = "good-grid-col"
		}
	}

	return ws
}

// Return current week day list.
// Week can be corrected by weekOffset.
func getWeekDayList(weekOffset int64) []int64 {
	timeNow := time.Now()
	currentDayUnix := timeNow.Unix()
	_, timeZoneSecondOffset := timeNow.Zone()
	currentDay := (currentDayUnix + int64(timeZoneSecondOffset)) / (60 * 60 * 24)
	currentWeek := ((currentDay + 3) / 7) + weekOffset

	CWFirstDay := currentWeek*7 - 3
	WeekDayList := make([]int64, 0, 8)
	var i int64 = 0
	for i = 0; i < 7; i++ {
		WeekDayList = append(WeekDayList, CWFirstDay+i)
	}
	return WeekDayList
}
