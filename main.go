package main

import (
	"encoding/json"
	"os"
	"sort"
	"strconv"
	"strings"

	untis "UntisTui/untis"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/joho/godotenv"
)

type model struct {
	days      [5][]untis.NamedTimetableEntry
	dayNames  [5]string
	timeSlots []string
	timeMaps  [5]map[string]untis.NamedTimetableEntry
}

func (m model) Init() tea.Cmd {
	return nil
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "q", "ctrl+c":
			return m, tea.Quit
		}
	}
	return m, nil
}

func main() {
	godotenv.Load(".env")
	user := os.Getenv("USER")
	pass := os.Getenv("PASS")

	untis.Main(user, pass)

	p := tea.NewProgram(newModel())
	if _, err := p.Run(); err != nil {
		panic(err)
	}
}

func loadJSON(path string) []untis.NamedTimetableEntry {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil
	}
	var entries []untis.NamedTimetableEntry
	if err := json.Unmarshal(data, &entries); err != nil {
		return nil
	}
	return entries
}

func newModel() model {
	mon := loadJSON("timetableFilled_Monday.json")
	tue := loadJSON("timetableFilled_Tuesday.json")
	wed := loadJSON("timetableFilled_Wednesday.json")
	thu := loadJSON("timetableFilled_Thursday.json")
	fri := loadJSON("timetableFilled_Friday.json")

	days := [5][]untis.NamedTimetableEntry{mon, tue, wed, thu, fri}
	dayNames := [5]string{"Mon", "Tue", "Wed", "Thu", "Fri"}
	var allTimes []string
	for _, dayEntries := range days {
		for _, e := range dayEntries {
			allTimes = append(allTimes, e.StartTime)
		}
	}
	timeSlots := sortTimeStrings(allTimes)
	var timeMaps [5]map[string]untis.NamedTimetableEntry
	for i, entries := range days {
		timeMaps[i] = buildTimeMap(entries)
	}
	return model{
		days:      days,
		dayNames:  dayNames,
		timeSlots: timeSlots,
		timeMaps:  timeMaps,
	}
}

func (m model) View() string {
	if len(m.timeSlots) == 0 {
		return "No timetable data.\nPress q to quit"
	}
	timeColWidth := 6
	entryColWidth := 18

	// Styling
	timeStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("8")).Width(timeColWidth).Align(lipgloss.Right)
	headerStyle := lipgloss.NewStyle().Bold(true).Padding(0, 1).Width(entryColWidth)
	entryStyle := lipgloss.NewStyle().Padding(0, 1).Width(entryColWidth).BorderStyle(lipgloss.RoundedBorder())

	headers := []string{timeStyle.Render("Time")}
	for _, name := range m.dayNames {
		headers = append(headers, headerStyle.Render(name))
	}
	rows := []string{lipgloss.JoinHorizontal(lipgloss.Top, headers...)}

	for _, timeSlot := range m.timeSlots {
		cells := []string{timeStyle.Render(timeSlot)}
		for dayIdx := 0; dayIdx < 5; dayIdx++ {
			if entry, exists := m.timeMaps[dayIdx][timeSlot]; exists {
				label := strings.Join(entry.Su, "/")
				if label == "" {
					label = "-"
				}
				cells = append(cells, entryStyle.Render(label))
			} else {
				cells = append(cells, entryStyle.Render(""))
			}
		}
		rows = append(rows, lipgloss.JoinHorizontal(lipgloss.Top, cells...))
	}
	content := lipgloss.JoinVertical(lipgloss.Left, rows...)
	return content + "\n\nPress q to quit"
}

func timeToMinutes(t string) int {
	parts := strings.Split(t, ":")
	h, _ := strconv.Atoi(parts[0])
	m, _ := strconv.Atoi(parts[1])
	return h*60 + m
}

func sortTimeStrings(times []string) []string {
	seen := make(map[string]bool)
	unique := []string{}
	for _, t := range times {
		if !seen[t] {
			seen[t] = true
			unique = append(unique, t)
		}
	}
	sort.Slice(unique, func(i, j int) bool {
		return timeToMinutes(unique[i]) < timeToMinutes(unique[j])
	})
	return unique
}

func buildTimeMap(entries []untis.NamedTimetableEntry) map[string]untis.NamedTimetableEntry {
	m := make(map[string]untis.NamedTimetableEntry)
	for _, e := range entries {
		m[e.StartTime] = e
	}
	return m
}
