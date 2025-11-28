package main

import (
	"encoding/json"
	"log"
	"os"
	"sort"
	"strconv"
	"strings"
	"time"

	untis "UntisTui/untis"

	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/joho/godotenv"
)

type model struct {
	days       [5][]untis.NamedTimetableEntry
	dayNames   [5]string
	timeSlots  []string
	timeMaps   [5]map[string]untis.NamedTimetableEntry
	viewport   viewport.Model
	width      int
	height     int
	currentDay int // 0=Monday, 1=Tuesday, etc.
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
		m.viewport.Height = msg.Height - 4 // account for title + status + footer + borders
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
		return m.renderEmptyState()
	}

	title := m.renderTitle()
	body := m.viewport.View()
	footer := m.renderFooter()

	// Create main container with consistent background
	mainStyle := lipgloss.NewStyle().
		Background(lipgloss.Color("#1F2937")). // Match table background
		Width(m.width).
		Height(m.height)

	content := lipgloss.JoinVertical(lipgloss.Top, title, body, footer)
	return mainStyle.Render(content)
}

func (m model) renderEmptyState() string {
	gradientStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("#FF6B6B")).
		Background(lipgloss.Color("#1F2937")). // Match table background
		Padding(8, 4).
		Align(lipgloss.Center).
		Width(m.width).
		Height(m.height)

	icon := "✨"
	title := "No Timetable Data Available"
	subtitle := "Please check your connection and credentials"
	help := "Press 'q' to quit"

	return gradientStyle.Render(
		lipgloss.JoinVertical(lipgloss.Center,
			lipgloss.NewStyle().Bold(true).MarginBottom(1).Render(icon),
			lipgloss.NewStyle().Bold(true).MarginBottom(1).Render(title),
			lipgloss.NewStyle().Foreground(lipgloss.Color("#A0A0A0")).Render(subtitle),
			lipgloss.NewStyle().MarginTop(2).Foreground(lipgloss.Color("#808080")).Render(help),
		),
	)
}

func (m model) renderTitle() string {
	// Gradient effect using multiple styles
	leftStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("#6366F1")).
		Background(lipgloss.Color("#1F2937")). // Match main background
		Bold(true).
		Padding(0, 1)

	middleStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("#8B5CF6")).
		Background(lipgloss.Color("#1F2937")). // Match main background
		Bold(true).
		Padding(0, 1)

	rightStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("#A855F7")).
		Background(lipgloss.Color("#1F2937")). // Match main background
		Bold(true).
		Padding(0, 1)

	calendar := leftStyle.Render("📅")
	title := middleStyle.Render("Weekly Timetable")
	books := rightStyle.Render("📚")

	return lipgloss.JoinHorizontal(lipgloss.Center, calendar, title, books)
}

func (m model) renderFooter() string {
	// Modern footer with subtle styling
	footerStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("#9CA3AF")).
		Background(lipgloss.Color("#1F2937")). // Match main background
		MarginTop(1).
		Padding(0, 2).
		Italic(true).
		Align(lipgloss.Center).
		Width(m.width)

	controls := []string{
		"q",
		"↑/↓ scroll",
		"home/end",
		"pg up/down",
	}

	styledControls := []string{}
	for i, control := range controls {
		if i == 0 {
			// Highlight quit command
			styledControls = append(styledControls,
				lipgloss.NewStyle().
					Foreground(lipgloss.Color("#EF4444")).
					Bold(true).
					Render("Quit: "+control))
		} else {
			styledControls = append(styledControls,
				lipgloss.NewStyle().
					Foreground(lipgloss.Color("#6B7280")).
					Render(control))
		}
	}

	return footerStyle.Render(strings.Join(styledControls, "  •  "))
}

// Helper function to check if a lesson is cancelled
func isLessonCancelled(entry untis.NamedTimetableEntry) bool {
	if entry.Statflags != "" {
		// Check for cancelled flag - this might vary based on your WebUntis system
		// Common cancelled flags might be "cancelled", "c", "x", "-", "K" etc.
		cancelledFlags := []string{"cancelled", "c", "x", "-", "K"}
		for _, flag := range cancelledFlags {
			if strings.Contains(strings.ToLower(entry.Statflags), flag) {
				return true
			}
		}
	}
	// Also check if code contains cancellation indicators
	if entry.Code != "" {
		cancelledCodes := []string{"cancelled", "canceled", "entfällt", "ausfall"}
		for _, code := range cancelledCodes {
			if strings.Contains(strings.ToLower(entry.Code), code) {
				return true
			}
		}
	}
	return false
}

// Helper function to check if we need lunch break spacing (specifically between lesson 6 and 7)
func needsLunchBreakSpacing(currentTime string, allEntries []untis.NamedTimetableEntry, currentIndex int) bool {
	// Find the current lesson's end time and next lesson's start time
	var currentEndTime string
	var nextStartTime string

	// Find current lesson's end time
	for _, entry := range allEntries {
		if entry.StartTime == currentTime {
			currentEndTime = entry.EndTime
			break
		}
	}

	// Find next lesson's start time (if exists)
	if currentIndex < len(allEntries)-1 {
		nextStartTime = allEntries[currentIndex+1].StartTime
	}

	// Specifically check for the lunch break pattern: lesson ending at 13:05, next starting at 13:45
	if currentEndTime == "13:05" && nextStartTime == "13:45" {
		return true
	}

	return false
}

// Helper function to get lesson number based on time patterns
func getLessonNumber(startTime string) int {
	// Common school lesson patterns (this might need adjustment based on your specific schedule)
	lessonTimes := map[string]int{
		"08:00": 1, "08:40": 1, "08:45": 1,
		"09:25": 2, "09:30": 2,
		"09:40": 3, "10:25": 3,
		"10:30": 4, "11:15": 4,
		"11:30": 5, "12:15": 5,
		"12:20": 6, "13:05": 6,
		"13:45": 7, "14:30": 7,
		"15:15": 8,
		"15:25": 9, "16:10": 9,
		"16:55": 10,
	}

	if lessonNum, exists := lessonTimes[startTime]; exists {
		return lessonNum
	}
	return 0
}

// renderTableContent generates the timetable string (pure function)
func (m model) renderTableContent() string {
	if len(m.timeSlots) == 0 {
		return "No timetable data."
	}

	// Responsive breakpoints and sizing constants
	const (
		largeTerminalWidth  = 150
		mediumTerminalWidth = 110
		largeEntryWidth     = 24
		largeTimeWidth      = 12
		mediumEntryWidth    = 20
		mediumTimeWidth     = 10
		smallEntryWidth     = 18
		smallTimeWidth      = 9
		minRoomDisplayWidth = 20
		minCodeDisplayWidth = 20
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

	// Modern color palette
	primaryColor := lipgloss.Color("#6366F1")   // Indigo
	secondaryColor := lipgloss.Color("#8B5CF6") // Purple
	accentColor := lipgloss.Color("#EC4899")    // Pink
	textColor := lipgloss.Color("#F9FAFB")      // Almost white
	mutedColor := lipgloss.Color("#6B7280")     // Gray
	successColor := lipgloss.Color("#10B981")   // Emerald
	warningColor := lipgloss.Color("#F59E0B")   // Amber
	errorColor := lipgloss.Color("#EF4444")     // Red
	cancelledColor := lipgloss.Color("#6B7280") // Gray for cancelled

	// Enhanced time column styling with start and end times
	timeStrStyle := lipgloss.NewStyle().
		Foreground(primaryColor).
		Background(lipgloss.Color("#1F2937")). // Add background color
		Width(timeColWidth).
		Align(lipgloss.Center).
		Bold(true).
		Padding(0, 1).
		BorderStyle(lipgloss.Border{Left: "│", Right: "│"}).
		BorderForeground(lipgloss.Color("#374151"))

	timeStyle := lipgloss.NewStyle().
		PaddingTop(1).
		Foreground(secondaryColor).
		Background(lipgloss.Color("#1F2937")). // Add background color
		Width(timeColWidth).
		Align(lipgloss.Center).
		Bold(true).
		Padding(0, 1).
		BorderStyle(lipgloss.Border{Left: "│", Right: "│"}).
		BorderForeground(lipgloss.Color("#374151"))

	// Enhanced header styling with gradient effect
	headerStyle := lipgloss.NewStyle().
		Bold(true).
		Padding(1, 1).
		Width(entryColWidth+2).
		Align(lipgloss.Center).
		Foreground(textColor).
		Background(primaryColor).
		BorderStyle(lipgloss.Border{Left: "│", Right: "│"}).
		BorderForeground(lipgloss.Color("#374151")).
		Margin(0, 0, 1, 0)

	// Enhanced entry styling with hover-like effects
	entryStyle := lipgloss.NewStyle().
		Foreground(successColor).
		Bold(true).
		Padding(1, 1).
		Width(entryColWidth).
		BorderStyle(lipgloss.RoundedBorder()).
		BorderForeground(accentColor).
		BorderBackground(lipgloss.Color("#1F2937")).
		Background(lipgloss.Color("#1F2937")).
		Align(lipgloss.Center).
		Margin(0, 1)

	// Special styling for current day column
	currentDayHeaderStyle := lipgloss.NewStyle().
		Bold(true).
		Padding(1, 1).
		Width(entryColWidth+2).
		Align(lipgloss.Center).
		Foreground(lipgloss.Color("#F59E0B")). // Amber text for current day
		Background(lipgloss.Color("#78350F")). // Dark amber background
		BorderStyle(lipgloss.Border{Left: "│", Right: "│"}).
		BorderForeground(lipgloss.Color("#F59E0B")). // Amber border
		Margin(0, 0, 1, 0)

	// Special cancelled entry styling with strikethrough effect
	cancelledStyle := lipgloss.NewStyle().
		Foreground(cancelledColor).
		Bold(false).
		Padding(1, 1).
		Width(entryColWidth).
		BorderStyle(lipgloss.RoundedBorder()).
		BorderForeground(cancelledColor).
		BorderBackground(lipgloss.Color("#374151")).
		Background(lipgloss.Color("#374151")).
		Align(lipgloss.Center).
		Margin(0, 1).
		Strikethrough(true)

	// Enhanced empty entry styling
	emptyEntryStyle := lipgloss.NewStyle().
		Foreground(mutedColor).
		Padding(1, 1).
		Width(entryColWidth).
		BorderStyle(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("#374151")).
		BorderBackground(lipgloss.Color("#1F2937")).
		Background(lipgloss.Color("#1F2937")). // Fix background color
		Align(lipgloss.Center).
		Margin(0, 1)

	// Special styling for different types of entries
	examStyle := lipgloss.NewStyle().
		Foreground(errorColor).
		Bold(true).
		Padding(1, 1).
		Width(entryColWidth).
		BorderStyle(lipgloss.RoundedBorder()).
		BorderForeground(errorColor).
		BorderBackground(lipgloss.Color("#7F1D1D")).
		Background(lipgloss.Color("#7F1D1D")).
		Align(lipgloss.Center).
		Margin(0, 1)

	specialStyle := lipgloss.NewStyle().
		Foreground(warningColor).
		Bold(true).
		Padding(1, 1).
		Width(entryColWidth).
		BorderStyle(lipgloss.RoundedBorder()).
		BorderForeground(warningColor).
		BorderBackground(lipgloss.Color("#78350F")).
		Background(lipgloss.Color("#78350F")).
		Align(lipgloss.Center).
		Margin(0, 1)

	// Enhanced day icons with better Unicode characters
	dayIcons := map[string]string{
		"Mon": "",
		"Tue": "",
		"Wed": "",
		"Thu": "",
		"Fri": "",
	}

	// Create header row with current day highlighted
	headers := []string{timeStrStyle.Render("Time")}
	for i, name := range m.dayNames {
		icon := dayIcons[name]
		if i == m.currentDay {
			// Use highlighted style for current day
			headers = append(headers, currentDayHeaderStyle.Render(icon+" "+name))
		} else {
			// Use normal style for other days
			headers = append(headers, headerStyle.Render(icon+" "+name))
		}
	}
	// Join header cells with background color to prevent transparent gaps
	headerRowStyle := lipgloss.NewStyle().
		Background(lipgloss.Color("#1F2937")).
		Width(m.width)
	headerContent := lipgloss.JoinHorizontal(lipgloss.Center, headers...)
	rows := []string{headerRowStyle.Render(headerContent)}

	// Add separator line
	separator := lipgloss.NewStyle().
		Foreground(lipgloss.Color("#374151")).
		Width(m.width - 4).
		Render(strings.Repeat("─", m.width-4))
	rows = append(rows, separator)

	// Collect all entries for lunch break detection
	allEntries := []untis.NamedTimetableEntry{}
	for dayIdx := 0; dayIdx < 5; dayIdx++ {
		for _, entry := range m.days[dayIdx] {
			allEntries = append(allEntries, entry)
		}
	}

	// Sort entries by start time for proper lesson ordering
	sort.Slice(allEntries, func(i, j int) bool {
		return timeToMinutes(allEntries[i].StartTime) < timeToMinutes(allEntries[j].StartTime)
	})

	// Render timetable entries with enhanced time display
	for i, timeSlot := range m.timeSlots {
		// Show both start and end time
		timeDisplay := timeSlot
		if i < len(m.timeSlots)-1 {
			nextTimeSlot := m.timeSlots[i+1]
			timeDisplay = timeSlot + "-" + nextTimeSlot
		} else {
			// For the last slot, try to find end time from entries
			for dayIdx := 0; dayIdx < 5; dayIdx++ {
				if entry, exists := m.timeMaps[dayIdx][timeSlot]; exists && entry.EndTime != "" {
					timeDisplay = timeSlot + "-" + entry.EndTime
					break
				}
			}
		}

		cells := []string{timeStyle.Render(timeDisplay)}
		for dayIdx := 0; dayIdx < 5; dayIdx++ {
			if entry, exists := m.timeMaps[dayIdx][timeSlot]; exists {
				subject := strings.Join(entry.Su, "/")
				room := ""
				if len(entry.Ro) > 0 {
					room = entry.Ro[0]
				}
				code := ""
				if entry.Code != "" {
					code = entry.Code
				}

				var label string
				var style lipgloss.Style

				// Check if lesson is cancelled
				if isLessonCancelled(entry) {
					style = cancelledStyle
				} else if strings.Contains(strings.ToLower(code), "exam") || strings.Contains(strings.ToLower(code), "test") {
					style = examStyle
				} else if strings.Contains(strings.ToLower(code), "special") || strings.Contains(strings.ToLower(code), "event") {
					style = specialStyle
				} else {
					style = entryStyle
				}

				// Special styling for current day entries
				if dayIdx == m.currentDay {
					currentDayStyle := lipgloss.NewStyle().
						Foreground(lipgloss.Color("#FBBF24")). // Bright amber text for current day
						Bold(true).
						Padding(1, 1).
						Width(entryColWidth).
						BorderStyle(lipgloss.RoundedBorder()).
						BorderForeground(lipgloss.Color("#F59E0B")). // Amber border
						BorderBackground(lipgloss.Color("#1F2937")).
						Background(lipgloss.Color("#1F2937")). // normal background
						Align(lipgloss.Center).
						Margin(0, 1)

					// Use current day style for today's lessons
					if !isLessonCancelled(entry) {
						style = currentDayStyle
					}
				}

				if subject == "" {
					label = "Free Period"
				} else {
					displaySubject := subject
					maxSubjectLen := entryColWidth - minTextPadding
					if maxSubjectLen < 2 {
						maxSubjectLen = 2
					}
					if len(displaySubject) > maxSubjectLen {
						displaySubject = displaySubject[:maxSubjectLen-1] + "…"
					}

					// For cancelled lessons, just show the subject name (no "CANCELLED" tag)
					label = displaySubject

					// Add room information
					if room != "" && entryColWidth >= minRoomDisplayWidth {
						displayRoom := room
						maxRoomLen := entryColWidth - minTextPadding
						if maxRoomLen < 2 {
							maxRoomLen = 2
						}
						if len(displayRoom) > maxRoomLen {
							displayRoom = displayRoom[:maxRoomLen-1] + "…"
						}
						label += "\n🏫 " + displayRoom
					}

					// Add code information
					if code != "" && entryColWidth >= minCodeDisplayWidth {
						displayCode := code
						maxCodeLen := entryColWidth - minTextPadding
						if maxCodeLen < 2 {
							maxCodeLen = 2
						}
						if len(displayCode) > maxCodeLen {
							displayCode = displayCode[:maxCodeLen-1] + "…"
						}
						label += "\n" + displayCode
					}
				}
				cells = append(cells, style.Render(label))
			} else {
				// Special styling for current day free periods
				if dayIdx == m.currentDay {
					currentDayFreeStyle := lipgloss.NewStyle().
						Foreground(lipgloss.Color("#FBBF24")). // Bright amber text
						Padding(1, 1).
						Width(entryColWidth).
						BorderStyle(lipgloss.RoundedBorder()).
						BorderForeground(lipgloss.Color("#F59E0B")). // Amber border
						BorderBackground(lipgloss.Color("#1F2937")).
						Background(lipgloss.Color("#1F2937")).
						Align(lipgloss.Center).
						Margin(0, 1)

					cells = append(cells, currentDayFreeStyle.Render("Free"))
				} else {
					cells = append(cells, emptyEntryStyle.Render("Free"))
				}
			}
		}
		// Join cells with background color to prevent transparent gaps
		rowStyle := lipgloss.NewStyle().
			Background(lipgloss.Color("#1F2937")).
			Width(m.width)
		rowContent := lipgloss.JoinHorizontal(lipgloss.Center, cells...)
		rows = append(rows, rowStyle.Render(rowContent))

		// Add lunch break spacing ONLY between lesson 6 and 7
		// Specifically: when current lesson ends at 13:05, next starts at 13:45
		if i < len(m.timeSlots)-1 {
			currentTime := timeSlot
			nextTime := m.timeSlots[i+1]

			// Check if this is the specific lunch break pattern
			currentLessonNum := getLessonNumber(currentTime)
			nextLessonNum := getLessonNumber(nextTime)

			// Specifically check for lesson 6 ending and lesson 7 starting
			if currentLessonNum == 6 && nextLessonNum == 7 {
				// Verify the specific times to be sure
				var currentEndTime string
				var nextStartTime string

				// Find current lesson's end time
				for dayIdx := 0; dayIdx < 5; dayIdx++ {
					if entry, exists := m.timeMaps[dayIdx][currentTime]; exists {
						currentEndTime = entry.EndTime
						break
					}
				}

				// Next lesson's start time is the next time slot
				nextStartTime = nextTime

				// Only show lunch break for the specific 13:05 to 13:45 gap
				if currentEndTime == "13:05" && nextStartTime == "13:45" {
					// Add a lunch break separator
					lunchStyle := lipgloss.NewStyle().
						Foreground(lipgloss.Color("#F59E0B")).
						Background(lipgloss.Color("#78350F")).
						Padding(0, 2).
						Align(lipgloss.Center).
						Width(m.width-8).
						Margin(1, 4)

					lunchBreak := lunchStyle.Render("LUNCH BREAK")
					rows = append(rows, lunchBreak)

					// Add some vertical space
					rows = append(rows, "")
				}
			}
		}
	}

	// Join rows with background color to prevent transparent gaps
	tableContainerStyle := lipgloss.NewStyle().
		Background(lipgloss.Color("#1F2937")).
		Width(m.width)
	tableContent := tableContainerStyle.Render(lipgloss.JoinVertical(lipgloss.Center, rows...))

	// Enhanced border styling
	borderStyle := lipgloss.NewStyle().
		Border(lipgloss.ThickBorder()).
		BorderForeground(lipgloss.Color("#374151")).
		BorderBackground(lipgloss.Color("#1F2937")).
		Background(lipgloss.Color("#1F2937")).
		Padding(1, 2).
		Margin(1, 2).
		Width(m.width - 8)

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

	// Get current day of week (Monday=0, Tuesday=1, etc.)
	currentDay := int(time.Now().Weekday() - 1) // time.Weekday starts with Sunday=0
	if currentDay < 0 || currentDay > 4 {       // If it's Saturday (5) or Sunday (6), default to Monday
		currentDay = 0
	}

	// Initialize viewport with fallback size
	vp := viewport.New(80, 20)
	content := renderInitialTable(days, dayNames, timeSlots, timeMaps, currentDay)
	vp.SetContent(content)

	return model{
		days:       days,
		dayNames:   dayNames,
		timeSlots:  timeSlots,
		timeMaps:   timeMaps,
		viewport:   vp,
		width:      80,
		height:     24,
		currentDay: currentDay,
	}
}

// Helper to render initial table before WindowSizeMsg arrives
func renderInitialTable(days [5][]untis.NamedTimetableEntry, dayNames [5]string, timeSlots []string, timeMaps [5]map[string]untis.NamedTimetableEntry, currentDay int) string {
	// Create a temporary model-like struct to reuse render logic
	tempModel := model{
		days:       days,
		dayNames:   dayNames,
		timeSlots:  timeSlots,
		timeMaps:   timeMaps,
		width:      120, // reasonable default for initial render
		currentDay: currentDay,
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
