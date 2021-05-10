package goviewEcho

import (
	"fmt"
	"github.com/Sarraksh/OTRS-time-accounting/internal/httpServer"
	"github.com/foolin/goview"
	"github.com/foolin/goview/supports/echoview-v4"
	"github.com/labstack/echo"
	"github.com/labstack/echo/middleware"
	"net/http"
	"time"
)

// Implement httpServer Provider.
// Use "github.com/labstack/echo" as web engine and "github.com/foolin/goview" for work with http templates.
type Provider struct {
	Echo      *echo.Echo                 // Web engine instance.
	TodayData *httpServer.TodayStatistic // The structure to be filled in at the level of business logic.
}

// Initialise and return web service provider.
func NewProvider(todayData *httpServer.TodayStatistic, getCurrentWeekData, getLastWeekData func() (httpServer.WeekStatistic, error), setWDO, removeWDO chan string) httpServer.Provider {
	// Initial echo instance.
	e := echo.New()

	// Default middleware.
	e.Use(middleware.Logger())
	e.Use(middleware.Recover())

	// Set default renderer with custom template location.
	gvConf := goview.DefaultConfig
	gvConf.Root = "website" // Set template folder.
	e.Renderer = echoview.New(gvConf)

	// Set router schema.
	e = setPageRouter(e, todayData, getCurrentWeekData, getLastWeekData)
	e = setAPIRouter(e, setWDO, removeWDO)

	return Provider{Echo: e, TodayData: todayData}
}

// Initialise pages for web interface.
func setPageRouter(e *echo.Echo, todayData *httpServer.TodayStatistic, getCurrentWeekData, getLastWeekData func() (httpServer.WeekStatistic, error)) *echo.Echo {
	// Favicon.
	e.GET("/favicon.ico", wrapperFavIco())

	// Main page (today statistic).
	e.GET("/", func(c echo.Context) error {
		date := time.Now().Format("2006.02.01")
		timeNow := time.Now().Format("15:04:05")

		prodData, updateDateTime := todayData.Get()
		updateDateTimeText := updateDateTime.Format("2006.02.01 15:04:05")

		// Render with page master.html.
		return c.Render(http.StatusOK, "index", echo.Map{
			"title":          "Today",
			"date":           date,
			"time":           timeNow,
			"updateDateTime": updateDateTimeText,
			"prodData":       prodData,
		})
	})

	// Current week statistic page.
	e.GET("/currentweek", wrapperWeek(getCurrentWeekData, "Current week", "Списано за текущую неделю"))

	// Last week statistic page.
	e.GET("/lastweek", wrapperWeek(getLastWeekData, "Last week", "Списано за прошлую неделю"))

	return e
}

// Return handler function for week statistic render.
func wrapperWeek(getData func() (httpServer.WeekStatistic, error), title, pageName string) func(c echo.Context) error {
	return func(c echo.Context) error {
		dataTable, err := getData()
		if err != nil {
			// TODO - use error page template
			return c.String(http.StatusInternalServerError, fmt.Sprintf("Internal server error. Can't read statistic from internal storage.\n'%v'", err))
		}
		pageOpenTime := time.Now().Format("2006.02.01 15:04:05")

		//render with master
		return c.Render(http.StatusOK, "week", echo.Map{
			"title":        title,
			"pageName":     pageName,
			"pageOpenTime": pageOpenTime,
			"dataTable":    dataTable,
		})
	}
}

// Initialise web API.
func setAPIRouter(e *echo.Echo, setWDO, removeWDO chan string) *echo.Echo {
	// API.
	e.POST("/workingDayOverride", wrapperOverrideDay(setWDO))
	e.DELETE("/workingDayOverride", wrapperOverrideDay(removeWDO))

	return e
}

// Return handler function for add or remove day override.
func wrapperOverrideDay(output chan string) func(c echo.Context) error {
	return func(c echo.Context) error {
		output <- c.FormValue("day")
		return c.NoContent(http.StatusOK)
	}
}

// Return handler function for favicon.
func wrapperFavIco() func(c echo.Context) error {
	return func(c echo.Context) error {
		return c.File("website/favicon.ico")
	}
}

// Start HTTP server.
func (p Provider) ListenAndServe(port string) {
	port = fmt.Sprintf(":%s", port)
	p.Echo.Logger.Fatal(p.Echo.Start(port))
}
