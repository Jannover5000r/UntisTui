package untis

import (
	"net/http"
	"time"
)

func getMonday(t time.Time) time.Time {
	offset := int(t.Weekday() - time.Monday)
	if offset < 0 {
		offset += 7
	}
	return t.AddDate(0, 0, -offset)
}

func getWeekTable(cookies []*http.Cookie) {
	now := time.Now()
	monday := getMonday(now)
	tuesday := monday.AddDate(0, 0, 2)
	wednesday := monday.AddDate(0, 0, 2)
	thursday := monday.AddDate(0, 0, 3)
	friday := monday.AddDate(0, 0, 4)
	Weekday := "Monday"
	Timetable(cookies, monday, Weekday)
	Weekday = "Tuesday"
	Timetable(cookies, tuesday, Weekday)
	Weekday = "Wednesday"
	Timetable(cookies, wednesday, Weekday)
	Weekday = "Thursday"
	Timetable(cookies, thursday, Weekday)
	Weekday = "Friday"
	Timetable(cookies, friday, Weekday)
}

func TimetableWeek(cookies []*http.Cookie) {
	getWeekTable(cookies)
}
