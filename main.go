package main

import (
	"encoding/json"
	"log"
	"os"
	"sort"
	"strconv"
	"strings"

	untis "UntisTui/untis"

	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/joho/godotenv"
)

type model struct {
	days      [5][]untis.NamedTimetableEntry
	dayNames  [5]string
	timeSlots []string
	timeMaps  [5]map[string]untis.NamedTimetableEntry
	viewport  viewport.Model
	ready     bool
}

func (m model) Init() tea.Cmd {
	return tea.Batch(
		tea.EnterAltScreen,
		tea.HideCursor,
	)
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmds []tea.Cmd

	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "q", "ctrl+c", "esc":
			return m, tea.Sequence(
				tea.ShowCursor,
				tea.ExitAltScreen,
				tea.Quit,
			)
		}

	case tea.WindowSizeMsg:
		m.viewport.Width = msg.Width
		m.viewport.Height = msg.Height - 4
		if m.ready {
			m.viewport.SetContent(m.renderTable())
		}
	}

	vp, cmd := m.viewport.Update(msg)
	m.viewport = vp
	cmds = append(cmds, cmd)

	return m, tea.Batch(cmds...)
}

func main() {
	err := godotenv.Overload(".env")
	if err != nil {
		log.Println("error reading .env ", err)
	}
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
	if !m.ready {
		content := m.renderTable()
		m.viewport = viewport.New(160, 50) // placeholder; will be resized
		m.viewport.SetContent(content)
		m.ready = true
	}

	termWidth, termHeight := m.viewport.Width, m.viewport.Height
	if termWidth == 0 || termHeight == 0 {
		termWidth, termHeight = 80, 24
	}

	if len(m.timeSlots) == 0 {
		return "No timetable data.\nPress q to quit"
	}
	timeColWidth := 6
	entryColWidth := 18

	// Styling
	timeStrStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("63")).Width(timeColWidth).Align(lipgloss.Right)
	timeStyle := lipgloss.NewStyle().PaddingTop(1).Foreground(lipgloss.Color("63")).Width(timeColWidth).Align(lipgloss.Right)
	headerStyle := lipgloss.NewStyle().Bold(true).Padding(0, 1).Width(entryColWidth + 2).Align(lipgloss.Center).Foreground(lipgloss.Color("63"))
	entryStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("2")).Bold(true).Padding(0, 1).Width(entryColWidth).BorderStyle(lipgloss.RoundedBorder()).BorderForeground(lipgloss.Color("63")).Align(lipgloss.Center)

	headers := []string{timeStrStyle.Render("Time")}
	for _, name := range m.dayNames {
		headers = append(headers, headerStyle.Render(name))
	}
	rows := []string{lipgloss.JoinHorizontal(lipgloss.Top, headers...)}

	for _, timeSlot := range m.timeSlots {

		cells := []string{timeStyle.Render(timeSlot)}
		for dayIdx := 0; dayIdx < 5; dayIdx++ {
			if entry, exists := m.timeMaps[dayIdx][timeSlot]; exists {
				var lines []string

				if len(entry.Su) > 0 {
					lines = append(lines, strings.Join(entry.Su, "/"))
				} else {
					lines = append(lines, "-")
				}

				if len(entry.Ro) > 0 {
					lines = append(lines, strings.Join(entry.Ro, "/"))
				}

				if entry.Code != "" {
					lines = append(lines, entry.Code)
				}

				label := strings.Join(lines, "\n")
				cells = append(cells, entryStyle.Render(label))
			} else {
				cells = append(cells, entryStyle.Render(""))
			}
		}
		rows = append(rows, lipgloss.JoinHorizontal(lipgloss.Top, cells...))
	}
	tableContent := m.renderTable()
	m.viewport.Width = termWidth
	m.viewport.Height = termHeight - 4 // leave room for title + footer
	m.viewport.SetContent(tableContent)

	title := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("63")).Padding(0, 1).Render("Weekly Timetable")

	body := m.viewport.View()
	footer := "\n↑/↓: scroll • q: quit"
	return lipgloss.JoinVertical(lipgloss.Top, title, body, footer)
}

func (m model) renderTable() string {
	if len(m.timeSlots) == 0 {
		return "No timetable data availible."
	}

	// Estimate column widths based on terminal (optional: hardcode for now)
	entryColWidth := 16
	timeColWidth := 6

	timeStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("8")).Width(timeColWidth).Align(lipgloss.Right)
	headerStyle := lipgloss.NewStyle().Bold(true).Padding(0, 1).Width(entryColWidth)
	entryStyle := lipgloss.NewStyle().
		Padding(0, 1).
		Width(entryColWidth).
		BorderStyle(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("63"))

	// Header
	headers := []string{timeStyle.Render("Time")}
	for _, name := range m.dayNames {
		headers = append(headers, headerStyle.Render(name))
	}
	rows := []string{lipgloss.JoinHorizontal(lipgloss.Top, headers...)}

	// Data rows
	for _, timeSlot := range m.timeSlots {
		cells := []string{timeStyle.Render(timeSlot)}
		for dayIdx := 0; dayIdx < 5; dayIdx++ {
			if entry, exists := m.timeMaps[dayIdx][timeSlot]; exists {
				var lines []string
				if len(entry.Su) > 0 {
					lines = append(lines, strings.Join(entry.Su, "/"))
				} else {
					lines = append(lines, "—")
				}
				if len(entry.Ro) > 0 {
					lines = append(lines, strings.Join(entry.Ro, "/"))
				}
				if entry.Code != "" {
					lines = append(lines, entry.Code)
				}
				label := strings.Join(lines, "\n")
				cells = append(cells, entryStyle.Render(label))
			} else {
				cells = append(cells, entryStyle.Render(""))
			}
		}
		rows = append(rows, lipgloss.JoinHorizontal(lipgloss.Top, cells...))
	}

	return lipgloss.JoinVertical(lipgloss.Left, rows...)
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
