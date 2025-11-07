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
	width     int
	height    int
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
		m.width = msg.Width
		m.height = msg.Height
		// Update viewport size and re-render content
		m.viewport.Width = msg.Width
		m.viewport.Height = msg.Height - 6 // account for title + footer + borders
		content := m.renderTableContent()
		m.viewport.SetContent(content)
	}

	// Forward messages to the viewport (essential for scrolling!)
	vp, cmd := m.viewport.Update(msg)
	m.viewport = vp
	cmds = append(cmds, cmd)

	return m, tea.Batch(cmds...)
}

func (m model) View() string {
	if len(m.timeSlots) == 0 {
		return "📅 No timetable data.\n\n󰌑  Press q to quit"
	}

	titleStyle := lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("15")).
		Background(lipgloss.Color("12")).
		Padding(0, 2).
		MarginBottom(1)

	title := titleStyle.Render("📅  Weekly Timetable  📚")

	body := m.viewport.View()

	footerStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("240")).
		MarginTop(1).
		Italic(true)

	footer := footerStyle.Render("󰌑  Press 'q' to quit  │  ↑/↓: scroll")

	return lipgloss.JoinVertical(lipgloss.Top, title, body, footer)
}

// renderTableContent generates the timetable string (pure function)
func (m model) renderTableContent() string {
	if len(m.timeSlots) == 0 {
		return "No timetable data."
	}

	// Responsive breakpoints and sizing constants
	const (
		largeTerminalWidth  = 140
		mediumTerminalWidth = 100
		largeEntryWidth     = 20
		largeTimeWidth      = 8
		mediumEntryWidth    = 16
		mediumTimeWidth     = 7
		smallEntryWidth     = 14
		smallTimeWidth      = 6
		minRoomDisplayWidth = 16
		minCodeDisplayWidth = 16
		minTextPadding      = 4
	)

	timeColWidth := largeTimeWidth
	entryColWidth := largeEntryWidth

	if m.width > 0 && m.width < largeTerminalWidth {
		entryColWidth = mediumEntryWidth
		timeColWidth = mediumTimeWidth
	}
	if m.width > 0 && m.width < mediumTerminalWidth {
		entryColWidth = smallEntryWidth
		timeColWidth = smallTimeWidth
	}

	primaryColor := lipgloss.Color("12")
	secondaryColor := lipgloss.Color("14")
	accentColor := lipgloss.Color("13")
	textColor := lipgloss.Color("15")
	mutedColor := lipgloss.Color("240")
	successColor := lipgloss.Color("10")

	timeStrStyle := lipgloss.NewStyle().
		Foreground(primaryColor).
		Width(timeColWidth).
		Align(lipgloss.Center).
		Bold(true)

	timeStyle := lipgloss.NewStyle().
		PaddingTop(1).
		Foreground(secondaryColor).
		Width(timeColWidth).
		Align(lipgloss.Center).
		Bold(true)

	headerStyle := lipgloss.NewStyle().
		Bold(true).
		Padding(0, 1).
		Width(entryColWidth + 2).
		Align(lipgloss.Center).
		Foreground(textColor).
		Background(primaryColor)

	entryStyle := lipgloss.NewStyle().
		Foreground(successColor).
		Bold(true).
		Padding(1, 1).
		Width(entryColWidth).
		BorderStyle(lipgloss.RoundedBorder()).
		BorderForeground(accentColor).
		Align(lipgloss.Center)

	emptyEntryStyle := lipgloss.NewStyle().
		Foreground(mutedColor).
		Padding(1, 1).
		Width(entryColWidth).
		BorderStyle(lipgloss.RoundedBorder()).
		BorderForeground(mutedColor).
		Align(lipgloss.Center)

	dayIcons := map[string]string{
		"Mon": "󰃭",
		"Tue": "󰃮",
		"Wed": "󰃯",
		"Thu": "󰃰",
		"Fri": "󰃱",
	}

	headers := []string{timeStrStyle.Render("  " + "Time")}
	for _, name := range m.dayNames {
		icon := dayIcons[name]
		headers = append(headers, headerStyle.Render(icon+" "+name))
	}
	rows := []string{lipgloss.JoinHorizontal(lipgloss.Top, headers...)}

	for _, timeSlot := range m.timeSlots {
		cells := []string{timeStyle.Render("  " + timeSlot)}
		for dayIdx := 0; dayIdx < 5; dayIdx++ {
			if entry, exists := m.timeMaps[dayIdx][timeSlot]; exists {
				subject := strings.Join(entry.Su, "/")
				room := ""
				if len(entry.Ro) > 0 {
					room = entry.Ro[0]
				}
				code := " "
				if entry.Code != "" {
					code = entry.Code
				}

				var label string
				if subject == "" {
					label = "─"
				} else {
					displaySubject := subject
					maxSubjectLen := entryColWidth - minTextPadding
					if maxSubjectLen < 2 {
						maxSubjectLen = 2
					}
					if len(displaySubject) > maxSubjectLen {
						displaySubject = displaySubject[:maxSubjectLen-1] + "…"
					}
					label = "  " + displaySubject
					if room != "" && entryColWidth >= minRoomDisplayWidth {
						displayRoom := room
						maxRoomLen := entryColWidth - minTextPadding
						if maxRoomLen < 2 {
							maxRoomLen = 2
						}
						if len(displayRoom) > maxRoomLen {
							displayRoom = displayRoom[:maxRoomLen-1] + "…"
						}
						label += "\n 󰍉 " + displayRoom
					}
					if code != "" && entryColWidth >= minCodeDisplayWidth {
						displayCode := code
						maxCodeLen := entryColWidth - minTextPadding
						if maxCodeLen < 2 {
							maxCodeLen = 2
						}
						if len(displayCode) > maxCodeLen {
							displayCode = displayCode[:maxCodeLen-1] + "…"
						}
						label += "\n " + displayCode
					}
				}
				cells = append(cells, entryStyle.Render(label))
			} else {
				cells = append(cells, emptyEntryStyle.Render("━"))
			}
		}
		rows = append(rows, lipgloss.JoinHorizontal(lipgloss.Top, cells...))
	}

	tableContent := lipgloss.JoinVertical(lipgloss.Left, rows...)

	// Apply border around table content
	borderStyle := lipgloss.NewStyle().
		Border(lipgloss.ThickBorder()).
		BorderForeground(primaryColor).
		Padding(1, 2)

	return borderStyle.Render(tableContent)
}

func main() {
	err := godotenv.Overload(".env")
	if err != nil {
		log.Println("error reading .env ", err)
	}
	user := os.Getenv("UNTIS_USERNAME")
	pass := os.Getenv("UNTIS_PASSWORD")
	url := os.Getenv("UNTIS_URL")
	untis.Main(user, pass, url)

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

	// Initialize viewport with fallback size
	vp := viewport.New(80, 20)
	content := renderInitialTable(days, dayNames, timeSlots, timeMaps)
	vp.SetContent(content)

	return model{
		days:      days,
		dayNames:  dayNames,
		timeSlots: timeSlots,
		timeMaps:  timeMaps,
		viewport:  vp,
		width:     80,
		height:    24,
	}
}

// Helper to render initial table before WindowSizeMsg arrives
func renderInitialTable(days [5][]untis.NamedTimetableEntry, dayNames [5]string, timeSlots []string, timeMaps [5]map[string]untis.NamedTimetableEntry) string {
	// Create a temporary model-like struct to reuse render logic
	tempModel := model{
		days:      days,
		dayNames:  dayNames,
		timeSlots: timeSlots,
		timeMaps:  timeMaps,
		width:     120, // reasonable default for initial render
	}
	return tempModel.renderTableContent()
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
