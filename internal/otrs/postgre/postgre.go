package postgre

import (
	"database/sql"
	"errors"
	"fmt"
	"github.com/Sarraksh/OTRS-time-accounting/internal/otrs"
	_ "github.com/lib/pq"
	"time"
)

const (
	dayFormatLayout = "2006_01_02" // Day format layout used in queries.

	// Get accounted time and ticket statistic.
	getTodayDataQuery = `
select
	users.last_name,
	coalesce(ta.sum, 0) as time,
	t."NotClosed",
	t."Locked",
	t."Open"
from
	(select create_by, sum(time_unit)::numeric::integer
		from time_accounting
		where
				create_time >= to_date('%s', 'YYYY_MM_DD')
			and create_time <  to_date('%s', 'YYYY_MM_DD')
		group by create_by
	) as ta
	right outer join
		users on ta.create_by = users.id
	left join
		(select
			user_id,
			count(*) as "Locked",
			count(*) filter (where ticket_state_id not in (2,3,9,15)) as "NotClosed",
			count(*) filter (where ticket_state_id = 4) as "Open"
			from ticket
			where
				ticket_lock_id != 1
		group by user_id
		) as t on t.user_id = users.id
	where last_name in (%s)
	order by last_name
;
`

	// Get accounted work time and overtime.
	getCustomDayDataQuery = `
select
	users.last_name,
	coalesce(sum(time_unit)::integer, 0) as "time",
	coalesce(overtime.value_int, 0) as "overTime"
from
	(select create_by, time_unit, article_id
		from time_accounting
		where
				create_time >= to_date('%s', 'YYYY_MM_DD')
			and create_time <  to_date('%s', 'YYYY_MM_DD')
	) as ta
	left join
	(SELECT object_id, value_int
		FROM public.dynamic_field_value
		where field_id = 87
 	) as overtime on overtime.object_id = ta.article_id 
	right outer join
		users on ta.create_by = users.id
	where last_name in (%s)
	group by last_name, overtime.value_int
	order by last_name
;
`
)

// Implement otrs Provider.
// DB is a connection to postgres DB that contains OTRS data.
type Postgre struct {
	DB       *sql.DB
	UserList string
}

// Row contain data for one user.
type OtrsDataRow struct {
	LastName             string
	TimeAccounted        int
	AllTicketCount       int
	NotClosedTicketCount int
	OpenTicketCount      int
}

// Initialise and return OTRS DB connector.
func NewProvider(host, port, user, password, dbName, sslMode string, userList []string) (Postgre, error) {
	// Construct DB connection string.
	dbConnectionString := fmt.Sprintf(
		"host=%s port=%s user=%s password=%s dbname=%s sslmode=%s",
		host, port, user, password, dbName, sslMode,
	)

	// Construct user list string for DB query.
	ul, err := constructUserList(userList)
	if err != nil {
		return Postgre{}, err
	}

	// Connect to DB.
	db, err := OpenDB(dbConnectionString)
	if err != nil {
		return Postgre{}, err
	}

	var p Postgre
	p.DB = db
	p.UserList = ul

	return p, nil
}

// Construct user list string for DB query.
func constructUserList(userList []string) (string, error) {
	// Return error if empty slice provided.
	if len(userList) < 1 {
		return "", errors.New("user list must contain at least one user")
	}

	// Construct user list string.
	formattedUserList := ""
	for _, user := range userList {
		formattedUserList = fmt.Sprintf("%s'%s',", formattedUserList, user)
	}

	// Truncate last comma.
	formattedUserList = formattedUserList[:len(formattedUserList)-1]

	return formattedUserList, nil
}

// Connect to DB.
func OpenDB(dbConnectionString string) (*sql.DB, error) {
	// open database
	db, err := sql.Open("postgres", dbConnectionString)
	if err != nil {
		return nil, err
	}

	// check db
	err = db.Ping()
	if err != nil {
		return nil, err
	}

	return db, nil
}

// Get today data from OTRS BD.
func (p Postgre) GetTodayData() ([]otrs.DayStatisticRow, error) {
	// Assemble Query string.
	today := time.Now().Format(dayFormatLayout)
	tomorrow := time.Now().Add(time.Hour * 24).Format(dayFormatLayout)
	query := fmt.Sprintf(getTodayDataQuery, today, tomorrow, p.UserList)

	// Query for data.
	rowList, err := p.DB.Query(query)
	if err != nil {
		return nil, err
	}
	defer rowList.Close()

	// Collect data from from query result.
	var lastName string
	var timeAccounted, notClosedTicketCount, lockedTicketCount, openTicketCount int
	var otrsData = make([]otrs.DayStatisticRow, 0, 32)
	for rowList.Next() {
		err = rowList.Scan(&lastName, &timeAccounted, &notClosedTicketCount, &lockedTicketCount, &openTicketCount)
		if err != nil {
			return nil, err
		}
		// Append current row.
		otrsData = append(otrsData,
			otrs.DayStatisticRow{
				LastName:             lastName,
				WorkTime:             timeAccounted,
				NotClosedTicketCount: notClosedTicketCount,
				LockedTicketCount:    lockedTicketCount,
				OpenTicketCount:      openTicketCount,
			})
	}

	return otrsData, nil
}

// Get accounted work time and overtime for specified day.
func (p Postgre) GetCustomDayData(day time.Time) ([]otrs.DayStatisticRow, error) {
	// Assemble Query string.
	dayStart := day.Format(dayFormatLayout)
	dayEnd := day.Add(time.Hour * 24).Format(dayFormatLayout)
	query := fmt.Sprintf(getCustomDayDataQuery, dayStart, dayEnd, p.UserList)

	// Query for data.
	rowList, err := p.DB.Query(query)
	if err != nil {
		return nil, err
	}
	defer rowList.Close()

	// Collect data from from query result.
	var lastName string
	var workTime, overTimeMark int
	var otrsData = make([]otrs.DayStatisticRow, 0, 32)
	for rowList.Next() {
		err = rowList.Scan(&lastName, &workTime, &overTimeMark)
		if err != nil {
			return nil, err
		}
		// Aggregate data based on the presence of the overtime mark.
		otrsData = addTime(otrsData, lastName, workTime, overTimeMark)
	}

	return otrsData, nil
}

// TODO - change DB query and remove current function.
// Aggregate data based on the presence of the overtime mark.
func addTime(data []otrs.DayStatisticRow, lastName string, time, overTimeMark int) []otrs.DayStatisticRow {
	// Define type of provided time.
	var workTime, overTime int
	if overTimeMark == 0 {
		workTime = time
		overTime = 0
	} else {
		workTime = 0
		overTime = time
	}

	// Avoid "index out of range" error.
	if len(data) == 0 {
		data = append(data,
			otrs.DayStatisticRow{
				LastName: lastName,
				WorkTime: workTime,
				OverTime: overTime,
			})
		return data
	}

	// If current LastName and LastName from last slice element are not equal, add new element, else update last element.
	lastElement := len(data) - 1
	if data[lastElement].LastName != lastName {
		data = append(data,
			otrs.DayStatisticRow{
				LastName: lastName,
				WorkTime: workTime,
				OverTime: overTime,
			})
	} else {
		data[lastElement].WorkTime = data[lastElement].WorkTime + workTime
		data[lastElement].OverTime = data[lastElement].OverTime + overTime
	}

	return data
}

// Close DB connection.
func (p Postgre) Stop() error {
	err := p.DB.Close()
	if err != nil {
		return err
	}
	return nil
}
