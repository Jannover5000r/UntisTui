package untis

import (
	"fmt"
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
	// now := time.Now()
	// monday := getMonday(now)
}

func main() {
	now := time.Now()
	monday := getMonday(now)
	mondaystr := monday.Format("20060102")
	fmt.Println("Monday is: ", mondaystr)
	tuesday := monday.AddDate(0, 0, 1)
	tuesdaystr := tuesday.Format("20060102")
	fmt.Println("Tuersday is: ", tuesdaystr)
}
