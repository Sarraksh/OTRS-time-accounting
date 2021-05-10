package service

import (
	"fmt"
	"github.com/Sarraksh/OTRS-time-accounting/internal/config"
	"github.com/Sarraksh/OTRS-time-accounting/internal/httpServer"
	"github.com/Sarraksh/OTRS-time-accounting/internal/httpServer/goviewEcho"
	"github.com/Sarraksh/OTRS-time-accounting/internal/internalDB"
	"github.com/Sarraksh/OTRS-time-accounting/internal/internalDB/gromSqlite3"
	"github.com/Sarraksh/OTRS-time-accounting/internal/otrs"
	"github.com/Sarraksh/OTRS-time-accounting/internal/otrs/postgre"
	"log"
	"time"
)

// Service contain all business logic, configuration and list of interfaces for external services.
// When start service initialise all interfaces and start goroutines.
type Service struct {
	Cfg  config.Config              // Configuration. Doesn't change after initialization.
	DB   internalDB.Provider        // Internal DB. Persistent storage.
	OTRS otrs.Provider              // Connection to OTRS data base. Contain SQL query templates.
	HTTP httpServer.Provider        // Shows data to users and has small API for insert some data.
	Data *httpServer.TodayStatistic // Struct uses to send data for display by HTTP server. Today statistic.
}

const (
	morningShiftColor = "morning-shift-grid-col" // Matches the color in the HTML template.
	eveningShiftColor = "evening-shift-grid-col" // Matches the color in the HTML template.
)

func Start() {
	var srv Service
	var err error

	// Read config from file.
	srv.Cfg, err = config.ReadConfigFromYAMLFile("config.yaml")
	if err != nil {
		log.Printf("Read config from file failed '%v'", err)
	}
	log.Printf("'%+v'\n", srv.Cfg)

	// Initialise internal DB.
	internalDBProvider, err := gromSqlite3.NewDB("")
	if err != nil {
		log.Printf("Internal DB initialisation error '%v'", err)
	}
	srv.DB = internalDBProvider

	// Initialise OTRS.
	otrsDB, err := postgre.NewProvider(
		srv.Cfg.OTRSConnection.Host,
		srv.Cfg.OTRSConnection.Port,
		srv.Cfg.OTRSConnection.UserName,
		srv.Cfg.OTRSConnection.Password,
		srv.Cfg.OTRSConnection.DBName,
		srv.Cfg.OTRSConnection.SSLMode,
		srv.userList(),
	)
	if err != nil {
		log.Printf("Otrs initialisation error '%v'", err)
	}
	srv.OTRS = otrsDB
	defer func() {
		err := srv.OTRS.Stop()
		if err != nil {
			log.Printf("Otrs stop error '%v'", err)
		}
	}()

	// Runs periodic data collection to display the web page.
	srv.Data = &httpServer.TodayStatistic{}
	go srv.RegularlyGetTodayData()

	// Read data from OTRS for last 20 days and store if into internal DB.
	err = srv.GetOldStatisticFromOTRS()
	if err != nil {
		log.Printf("get old data failed - '%v'", err)
	}

	// Start WorkdayOverrideWorker.
	// Wait for signal from web API and store received data into internal DB.
	newWorkdayOverride := make(chan string, 100)
	removeWorkdayOverride := make(chan string, 100)
	go srv.WDOWorker(newWorkdayOverride, removeWorkdayOverride)

	// Initialise pages and start HTTP server.
	srv.HTTP = goviewEcho.NewProvider(srv.Data, srv.WeekConnector(0), srv.WeekConnector(-1), newWorkdayOverride, removeWorkdayOverride)
	srv.HTTP.ListenAndServe(srv.Cfg.Web.Port)

}

// Handle requests for override workdays.
func (s *Service) WDOWorker(setWDO, removeWDO chan string) {
	var date string
	for {
		select {
		case date = <-setWDO:
			s.setWDO(date)
		case date = <-removeWDO:
			s.removeWDO(date)
		}
	}
}

// Add new overridden day into internal DB.
func (s *Service) setWDO(date string) {
	day, err := dateToUnixDay(date)
	if err != nil {
		log.Printf("can't set workday owerride '%v'", err)
		return
	}

	s.DB.SetWorkdayOverride(day)
}

// Remove overridden day internal DB.
func (s *Service) removeWDO(date string) {
	day, err := dateToUnixDay(date)
	if err != nil {
		log.Printf("can't remove workday owerride '%v'", err)
		return
	}

	s.DB.RemoveWorkdayOverride(day)
}

// Calculate day number since 1970.01.01 . 1970.01.01 day number is 0.
func dateToUnixDay(dateString string) (int64, error) {
	// Get time from string.
	// Explicitly specify the UTC time zone.
	date, err := time.Parse("2006.01.02 Z0700", fmt.Sprint(dateString, " Z"))
	if err != nil {
		return 0, err
	}

	// Calculate day number since 1970.01.01 .
	dayUnix := date.Unix() / (60 * 60 * 24)
	return dayUnix, nil
}

// Regularly (every 15 minutes) get today data from OTRS DB.
// Correct the time to synchronize with the quarters of an hour.
func (s *Service) RegularlyGetTodayData() {
	var min time.Duration
	var err error

	// Get initial data.
	err = s.UpdateTodayStatistic()
	if err != nil {
		log.Printf("Error wile update data '%v", err)
		return
	}

	// Correct time.
	min = time.Duration(60 - time.Now().Minute())
	time.Sleep((time.Minute % 15) * min)

	for {
		err = s.UpdateTodayStatistic()
		if err != nil {
			log.Printf("Error wile update data '%v", err)
			return
		}
		time.Sleep(time.Minute * 15)
	}
}

// Regularly (every day) get yesterday data from OTRS DB.
// Correct the time to synchronize with midnight.
func (s *Service) RegularlyGetYesterdayData() {
	var min, hour time.Duration
	var err error
	for {
		err = s.GetDayFromOTRSAndStore(0)
		if err != nil {
			log.Printf("Error wile get yestarday data from OTRS '%v", err)
			return
		}
		min = time.Duration(60 - time.Now().Minute())
		hour = time.Duration(23 - time.Now().Hour())
		time.Sleep(time.Minute*min + 5)
		time.Sleep(time.Hour * hour)
	}
}

// Collect data from OTRS and store in into variable.
// Used in today web page.
func (s *Service) UpdateTodayStatistic() error {
	// Update data into internal DB.
	// For actual statistic on current week page.
	err := s.GetDayFromOTRSAndStore(0)
	if err != nil {
		return err
	}

	// Get extended today data.
	OTRSData, err := s.OTRS.GetTodayData()
	if err != nil {
		return err
	}

	webDataList := make([]httpServer.TodayStatisticRow, 0, 32) // Initialise struct, represented web page table.

	// TODO - refactor data aggregation logic logic
	// Create slice for link between OTRS data and
	SliceLinkMap := make(map[string]int, 0)
	for id, row := range OTRSData {
		SliceLinkMap[row.LastName] = id
	}

	// Fill the table with collected data in certain order.
	for i, user := range getUserOrder() {
		currentRow := AssembleTodayRow(OTRSData, SliceLinkMap[user.LastName], user.LastName, user.WorkShiftColor)
		webDataList = append(webDataList, currentRow)
		if user.LastInGroup {
			webDataList[i].LastInGroup = true
		}
	}

	s.Data.Update(webDataList)

	return nil
}

func AssembleTodayRow(
	OTRSData []otrs.DayStatisticRow,
	otrsIndex int,
	lastName string,
	workShiftColor string,
) httpServer.TodayStatisticRow {
	// Calculate time cell color.
	var timeAccountedColor string
	switch {
	case OTRSData[otrsIndex].WorkTime+OTRSData[otrsIndex].OverTime < 240:
		timeAccountedColor = "bad-grid-col"
	case OTRSData[otrsIndex].WorkTime+OTRSData[otrsIndex].OverTime < 300:
		timeAccountedColor = "average-grid-col"
	default:
		timeAccountedColor = "good-grid-col"
	}

	// Assemble row data.
	return httpServer.TodayStatisticRow{
		LastName:           lastName,
		WorkShiftColor:     workShiftColor,
		TimeAccounted:      OTRSData[otrsIndex].WorkTime + OTRSData[otrsIndex].OverTime,
		TimeAccountedColor: timeAccountedColor,
		AllTicketCount:     OTRSData[otrsIndex].LockedTicketCount,
		ClosedTicketCount:  OTRSData[otrsIndex].LockedTicketCount - OTRSData[otrsIndex].NotClosedTicketCount,
		OpenTicketCount:    OTRSData[otrsIndex].OpenTicketCount,
		LastInGroup:        false,
	}
}

// TODO - get data from config file
func getUserOrder() []httpServer.UserCell {
	order := make([]httpServer.UserCell, 0, 32)

	order = append(order, httpServer.UserCell{LastName: "Вахрамеев", WorkShiftColor: morningShiftColor, LastInGroup: false})
	order = append(order, httpServer.UserCell{LastName: "Мостынец", WorkShiftColor: morningShiftColor, LastInGroup: false})
	order = append(order, httpServer.UserCell{LastName: "Рыжко", WorkShiftColor: morningShiftColor, LastInGroup: false})
	order = append(order, httpServer.UserCell{LastName: "Ельцов", WorkShiftColor: eveningShiftColor, LastInGroup: false})
	order = append(order, httpServer.UserCell{LastName: "Кирьяков", WorkShiftColor: eveningShiftColor, LastInGroup: false})
	order = append(order, httpServer.UserCell{LastName: "Кротов", WorkShiftColor: eveningShiftColor, LastInGroup: false})
	order = append(order, httpServer.UserCell{LastName: "Лапенков", WorkShiftColor: eveningShiftColor, LastInGroup: true})

	order = append(order, httpServer.UserCell{LastName: "Аманов", WorkShiftColor: morningShiftColor, LastInGroup: false})
	order = append(order, httpServer.UserCell{LastName: "Асланян", WorkShiftColor: morningShiftColor, LastInGroup: false})
	order = append(order, httpServer.UserCell{LastName: "Лантух", WorkShiftColor: morningShiftColor, LastInGroup: false})
	order = append(order, httpServer.UserCell{LastName: "Мехряков", WorkShiftColor: morningShiftColor, LastInGroup: false})
	order = append(order, httpServer.UserCell{LastName: "Техов", WorkShiftColor: morningShiftColor, LastInGroup: false})
	order = append(order, httpServer.UserCell{LastName: "Ермаков", WorkShiftColor: eveningShiftColor, LastInGroup: false})
	order = append(order, httpServer.UserCell{LastName: "Шенцов", WorkShiftColor: eveningShiftColor, LastInGroup: true})

	return order
}

// Read data from OTRS for last 20 days and store if into internal DB.
func (s *Service) GetOldStatisticFromOTRS() error {
	var err error
	var i int64
	for i = 0; i > -20; i-- {
		err = s.GetDayFromOTRSAndStore(i)
		if err != nil {
			return err
		}
	}
	return nil
}

// Get day statistic from OTRS and store accounted time into internal DB.
func (s *Service) GetDayFromOTRSAndStore(dayOffset int64) error {
	tmpTime := time.Now().Add((time.Hour * 24) * time.Duration(dayOffset)) // Add day offset to current time.
	OTRSData, err := s.OTRS.GetCustomDayData(tmpTime)                      // Collect target day data for from OTRS.
	if err != nil {
		return err
	}

	var unixDay int64
	unixDay, err = dateToUnixDay(tmpTime.Format("2006.01.02")) // Calculate day number.

	// Store collected data for every person.
	for _, row := range OTRSData {
		if err != nil {
			return err
		}
		s.DB.AddOrUpdateAccountedTime(row.LastName, unixDay, int64(row.WorkTime), int64(row.OverTime))
	}
	return nil
}

// Get slice of LastName from configured users.
func (s *Service) userList() []string {
	ul := make([]string, 0, 32)
	for _, user := range s.Cfg.UserList {
		ul = append(ul, user.LastName)
	}
	return ul
}
