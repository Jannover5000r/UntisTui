package untis

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"time"
)

type TimetableResponse struct {
	Jsonrpc string      `json:"jsonrpc"`
	ID      string      `json:"id"`
	Result  []timetable `json:"result"`
}
type NamedTimetableEntry struct {
	ID           int      `json:"id"`
	Date         string   `json:"date"`
	StartTime    string   `json:"startTime"`
	EndTime      string   `json:"endTime"`
	Code         string   `json:"code,omitempty"`
	Statflags    string   `json:"statflags,omitempty"`
	Kl           []string `json:"kl"`
	Su           []string `json:"su"`
	Ro           []string `json:"ro"`
	ActivityType string   `json:"activityType"`
}
type timetable struct {
	ID           int     `json:"id"`
	Date         int     `json:"date"`
	StartTime    int     `json:"startTime"`
	EndTime      int     `json:"endTime"`
	Code         string  `json:"code,omitempty"`
	Statflags    string  `json:"statflags,omitempty"`
	Kl           []IDObj `json:"kl"`
	Su           []IDObj `json:"su"`
	Ro           []IDObj `json:"ro"`
	ActivityType string  `json:"activityType"`
}
type IDObj struct {
	ID int `json:"id"`
}

type NamedObj struct {
	ID   int    `json:"id"`
	Name string `json:"name"`
}
type getTimetable struct {
	ID      string `json:"id"`
	Method  string `json:"method"`
	Params  params `json:"params"`
	Jsonrpc string `json:"jsonrpc"`
}
type params struct {
	StartDate string `json:"startDate"`
	EndDate   string `json:"endDate"`
	ID        int    `json:"id"`
	Type      int    `json:"type"`
}
type Loginresult struct {
	SessionID  string `json:"sessionId"`
	PersonType int    `json:"personType"`
	PersonID   int    `json:"personId"`
	KlasseID   int    `json:"klasseId"`
}

func ReadLoginResultFromFile(path string) (Loginresult, error) {
	var result Loginresult
	data, err := os.ReadFile(path)
	if err != nil {
		return result, err
	}
	err = json.Unmarshal(data, &result)
	return result, err
}

func Timetable(cookies []*http.Cookie, date time.Time, weekday string) {
	loginFile := "login.json"

	loginResult, err := ReadLoginResultFromFile(loginFile)
	if err != nil {
		log.Printf("Could not read login result for user %s: ", err)
		return
	}

	dateStr := date.Format("20060102")
	g := getTimetable{"If you read this, Hello", "getTimetable", params{dateStr, dateStr, loginResult.PersonID, loginResult.PersonType}, "2.0"}
	timetablesJSON, err := json.Marshal(g)
	if err != nil {
		log.Printf("Error marshaling timetable request: %v", err)
		return
	}
	timetable := bytes.NewReader(timetablesJSON)

	req, err := http.NewRequest("GET", URL, timetable)
	if err != nil {
		log.Printf("Error creating timetable request: %v", err)
		return
	}
	req.Header.Set("Content-Type", "application/json")
	for _, cookie := range cookies {
		req.AddCookie(cookie)
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		log.Printf("Error fetching timetable: %v", err)
		return
	}
	defer resp.Body.Close()

	response, err := io.ReadAll(resp.Body)
	if err != nil {
		log.Printf("Error reading timetable response: %v", err)
		return
	}

	var Response TimetableResponse
	if err := json.Unmarshal(response, &Response); err != nil {
		log.Printf("Error unmarshaling timetable response: %v", err)
		return
	}

	data, err := json.MarshalIndent(Response.Result, "", "  ")
	if err != nil {
		log.Printf("Error marshaling timetable result: %v", err)
		return
	}

	timetableFileTmp := "timetableTmp.json"

	if err := os.WriteFile(timetableFileTmp, data, 0o644); err != nil {
		log.Printf("Error writing timetable file: %v", err)
		return
	}

	log.Printf("Updated timetable for user ")
	setTimetable(weekday)
}

func LoadIDMap(path string) (map[int]string, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var objs []NamedObj
	if err := json.Unmarshal(data, &objs); err != nil {
		return nil, err
	}
	m := make(map[int]string)
	for _, obj := range objs {
		m[obj.ID] = obj.Name
	}
	return m, nil
}

func LoadTimetable(path string) ([]timetable, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var entries []timetable
	if err := json.Unmarshal(data, &entries); err != nil {
		return nil, err
	}
	return entries, nil
}

func formatTime(t int) string {
	h := t / 100
	m := t % 100
	return fmt.Sprintf("%02d:%02d", h, m)
}

func formatDate(date int) string {
	s := fmt.Sprintf("%08d", date) // ensures leading zeros
	year := s[:4]
	month := s[4:6]
	day := s[6:8]
	return fmt.Sprintf("%s-%s-%s", day, month, year)
}

func setTimetable(weekday string) {
	subjects, _ := LoadIDMap("subjects.json")
	rooms, _ := LoadIDMap("rooms.json")
	classes, _ := LoadIDMap("classes.json")

	timetableFileTmp := "timetableTmp.json"

	timetableTmp, _ := LoadTimetable(timetableFileTmp)

	var namedTimetable []NamedTimetableEntry
	for _, lesson := range timetableTmp {
		var klNames, suNames, roNames []string
		for _, kl := range lesson.Kl {
			klNames = append(klNames, classes[kl.ID])
		}
		for _, su := range lesson.Su {
			suNames = append(suNames, subjects[su.ID])
		}
		for _, ro := range lesson.Ro {
			roNames = append(roNames, rooms[ro.ID])
		}
		namedTimetable = append(namedTimetable, NamedTimetableEntry{
			ID:           lesson.ID,
			Date:         formatDate(lesson.Date),
			StartTime:    formatTime(lesson.StartTime),
			EndTime:      formatTime(lesson.EndTime),
			Code:         lesson.Code,
			Statflags:    lesson.Statflags,
			Kl:           klNames,
			Su:           suNames,
			Ro:           roNames,
			ActivityType: lesson.ActivityType,
		})
	}

	data, err := json.MarshalIndent(namedTimetable, "", "  ")
	if err != nil {
		log.Printf("Error marshaling named timetable: %v", err)
		return
	}

	timetableFilledFileWeekday := "timetableFilled_" + weekday + ".json"

	if err := os.WriteFile(timetableFilledFileWeekday, data, 0o644); err != nil {
		log.Printf("Error writing timetableFilled file: %v", err)
		return
	}
	log.Printf("Filled timetable for user")

	// remove temporary timetable file
	os.Remove("timetableTmp.json")
	log.Printf("Temporary timetable file removed")
}
