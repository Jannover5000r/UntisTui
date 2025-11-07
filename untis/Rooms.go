package untis

import (
	"bytes"
	"encoding/json"
	"io"
	"log"
	"net/http"
	"os"
)

type getRooms struct {
	ID      string `json:"id"`
	Method  string `json:"method"`
	Params  any    `json:"params"`
	Jsonrpc string `json:"jsonrpc"`
}

type RoomsResponse struct {
	Jsonrpc string `json:"jsonrpc"`
	ID      string `json:"id"`
	Result  []Room `json:"result"`
}

type Room struct {
	ID       int    `json:"id"`
	Name     string `json:"name"`
	LongName string `json:"longName"`
	Active   bool   `json:"active"`
	Building string `json:"building"`
}

func Rooms(cookies []*http.Cookie, url string) {
	// log.Println("Abrufen der Stunden")
	g := getRooms{"2023-05-06 15:44:22.215292", "getRooms", map[string]any{}, "2.0"}
	roomsJSON, err := json.Marshal(g)
	if err != nil {
		log.Fatalf("Error marshaling login data: %v", err)
		return
	}
	rooms := bytes.NewReader(roomsJSON)

	prompt, err := http.NewRequest("POST", url, rooms)
	if err != nil {
		log.Fatalf("Error creatingrequest: %v", err)
		return
	}
	// log.Println("prompt without extra header or cookie ", prompt)
	// log.Println("Cookie: ", cookies)

	prompt.Header.Set("Content-Type", "application/json")
	prompt.Header.Set("User-Agent", "Webuntis Test")

	for _, cookie := range cookies {
		// if cookie.Name == "JSESSIONID" {
		prompt.AddCookie(cookie)
		//	log.Printf("Added JSESSIONID cookie: %s=%s", cookie.Name, cookie.Value)
		//}
	}
	// log.Println("Request JSON:", string(roomsJson))
	out, err := http.DefaultClient.Do(prompt)
	if err != nil {
		log.Printf("Error during request: %v", err)
		return
	}
	defer out.Body.Close()
	// log.Println(out.Status)
	response, err := io.ReadAll(out.Body)
	if err != nil {
		log.Printf("Error reading response body: %v", err)
		return
	}
	// responseString := string(response)
	// log.Println("Repsonse ", responseString)
	var Response RoomsResponse
	err = json.Unmarshal(response, &Response)
	if err != nil {
		log.Fatalf("Error unmarshaling response: %v", err)
	}

	data, err := json.MarshalIndent(Response.Result, "", "  ")
	if err != nil {
		log.Fatal(err)
	}
	err = os.WriteFile("rooms.json", data, 0o644)
	if err != nil {
		log.Fatal(err)
	}
	log.Println("Updated Rooms")
}
