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

func getWeekTable(cookies []*http.Cookie, url string) {
	now := time.Now()
	monday := getMonday(now)
	tuesday := monday.AddDate(0, 0, 1)
	wednesday := monday.AddDate(0, 0, 2)
	thursday := monday.AddDate(0, 0, 3)
	friday := monday.AddDate(0, 0, 4)
	Weekday := "Monday"
	Timetable(cookies, monday, Weekday, url)
	Weekday = "Tuesday"
	Timetable(cookies, tuesday, Weekday, url)
	Weekday = "Wednesday"
	Timetable(cookies, wednesday, Weekday, url)
	Weekday = "Thursday"
	Timetable(cookies, thursday, Weekday, url)
	Weekday = "Friday"
	Timetable(cookies, friday, Weekday, url)
}

func TimetableWeek(cookies []*http.Cookie, url string) {
	getWeekTable(cookies, url)
}
