// Package untis: used for basic timetable managment and api calls
package untis

import (
	"bytes"
	"encoding/json"
	"io"
	"log"
	"net/http"
	"os"

	"github.com/joho/godotenv"
)

type Params struct {
	Users    string `json:"user"`
	Password string `json:"password"`
	Client   string `json:"client"`
}

type Login struct {
	ID     string `json:"id"`
	Method string `json:"method"`
	Params Params `json:"params"`

	Jsonrpc string `json:"jsonrpc"`
}
type LoginResponse struct {
	Jsonrpc string      `json:"jsonrpc"`
	ID      string      `json:"id"`
	Result  Loginresult `json:"result"`
}

func init() {
	godotenv.Overload("../.env")
}

var URL = "https://thalia.webuntis.com/WebUntis/jsonrpc.do?school=Mons_Tabor"

func Main(user string, password string) {
	godotenv.Load("../.env")
	cookies, err := Auth(user, password)
	if err != nil {
		log.Printf("Authentication failed for user %s: %v", user, err)
		return
	}
	Rooms(cookies)
	Classes(cookies)
	Subjects(cookies)
	TimetableWeek(cookies)
	Teachers(cookies)
}

func Auth(user string, password string) ([]*http.Cookie, error) {
	l := Login{"2023-05-06 15:44:22.215292", "authenticate", Params{user, password, "WebUntis Test"}, "2.0"}
	loginJSON, err := json.Marshal(l)
	if err != nil {
		return nil, err
	}
	login := bytes.NewReader(loginJSON)

	LoginOut, err := http.Post(URL, "application/json", login)
	if err != nil {
		return nil, err
	}
	defer LoginOut.Body.Close()

	cookies := LoginOut.Cookies()
	log.Printf("Login successful for user: %s", user)

	response, err := io.ReadAll(LoginOut.Body)
	if err != nil {
		return nil, err
	}

	var Response LoginResponse
	if err := json.Unmarshal(response, &Response); err != nil {
		return nil, err
	}

	data, err := json.MarshalIndent(Response.Result, "", "  ")
	if err != nil {
		return nil, err
	}

	loginFile := "login.json"

	if err := os.WriteFile(loginFile, data, 0o644); err != nil {
		return nil, err
	}

	return cookies, nil
}
