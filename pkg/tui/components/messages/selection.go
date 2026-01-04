package messages

import (
	"strings"
	"time"

	tea "charm.land/bubbletea/v2"
	"github.com/charmbracelet/x/ansi"
	"github.com/mattn/go-runewidth"

	"github.com/docker/cagent/pkg/tui/styles"
)

// selectionState encapsulates all state related to text selection
type selectionState struct {
	active          bool
	startLine       int
	startCol        int
	endLine         int
	endCol          int
	mouseButtonDown bool
	mouseY          int // Screen Y coordinate for autoscroll

	// Double-click detection
	lastClickTime time.Time
	lastClickLine int
	lastClickCol  int
}

// start initializes a new selection at the given position
func (s *selectionState) start(line, col int) {
	s.active = true
	s.mouseButtonDown = true
	s.startLine = line
	s.startCol = col
	s.endLine = line
	s.endCol = col
}

// update updates the end position of the selection
func (s *selectionState) update(line, col int) {
	s.endLine = line
	s.endCol = col
}

// end finalizes the selection and stops mouse tracking
func (s *selectionState) end() {
	s.mouseButtonDown = false
}

// clear resets all selection state
func (s *selectionState) clear() {
	*s = selectionState{}
}

// normalized returns the selection bounds in normalized order (start <= end)
func (s *selectionState) normalized() (startLine, startCol, endLine, endCol int) {
	startLine, startCol = s.startLine, s.startCol
	endLine, endCol = s.endLine, s.endCol

	if startLine > endLine || (startLine == endLine && startCol > endCol) {
		startLine, endLine = endLine, startLine
		startCol, endCol = endCol, startCol
	}
	return startLine, startCol, endLine, endCol
}

// isDoubleClick checks if a click at the given position constitutes a double-click
func (s *selectionState) isDoubleClick(line, col int) bool {
	if s.lastClickTime.IsZero() {
		return false
	}

	now := time.Now()
	colDiff := col - s.lastClickCol

	return now.Sub(s.lastClickTime) < 500*time.Millisecond &&
		line == s.lastClickLine &&
		colDiff >= -1 && colDiff <= 1
}

// recordClick stores click information for double-click detection
func (s *selectionState) recordClick(line, col int) {
	s.lastClickTime = time.Now()
	s.lastClickLine = line
	s.lastClickCol = col
}

// resetDoubleClick clears double-click detection state
func (s *selectionState) resetDoubleClick() {
	s.lastClickTime = time.Time{}
}

// AutoScrollTickMsg triggers auto-scroll during selection
type AutoScrollTickMsg struct {
	Direction int // -1 for up, 1 for down
}

// autoScroll handles automatic scrolling when selecting near viewport edges
func (m *model) autoScroll() tea.Cmd {
	const scrollThreshold = 2
	direction := 0

	// Use stored screen Y coordinate to check if mouse is in autoscroll region
	// mouseToLineCol subtracts 2 for header, so viewport-relative Y is mouseY - 2
	viewportY := max(m.selection.mouseY-2, 0)

	if viewportY < scrollThreshold && m.scrollOffset > 0 {
		// Scroll up - mouse is near top of viewport
		direction = -1
		m.scrollUp()
		// Update endLine to reflect new scroll position
		m.selection.endLine = max(0, m.selection.endLine-1)
	} else if viewportY >= m.height-scrollThreshold && viewportY < m.height {
		// Scroll down - mouse is near bottom of viewport
		maxScrollOffset := max(0, m.totalHeight-m.height)
		if m.scrollOffset < maxScrollOffset {
			direction = 1
			m.scrollDown()
			// Update endLine to reflect new scroll position
			m.selection.endLine++
		}
	}

	if direction == 0 {
		return nil
	}

	return tea.Tick(20*time.Millisecond, func(time.Time) tea.Msg {
		return AutoScrollTickMsg{Direction: direction}
	})
}

// selectWordAt selects the word at the given line and column position
func (m *model) selectWordAt(line, col int) {
	lines := strings.Split(m.rendered, "\n")
	if line < 0 || line >= len(lines) {
		return
	}

	originalLine := lines[line]
	plainLine := stripBorderChars(ansi.Strip(originalLine))
	if plainLine == "" {
		return
	}

	// Calculate border offset to adjust column position
	borderOffset := runewidth.StringWidth(ansi.Strip(originalLine)) - runewidth.StringWidth(plainLine)
	runes := []rune(plainLine)

	// Convert display column to rune index
	runeIdx := min(max(0, displayWidthToRuneIndex(plainLine, max(0, col-borderOffset))), len(runes)-1)
	if runeIdx < 0 {
		return
	}

	// Find word boundaries - determine if we're on a word or non-word char
	onWordChar := isWordChar(runes[runeIdx])
	startIdx, endIdx := runeIdx, runeIdx

	// Expand to find contiguous characters of the same type
	for startIdx > 0 && isWordChar(runes[startIdx-1]) == onWordChar {
		startIdx--
	}
	for endIdx < len(runes)-1 && isWordChar(runes[endIdx+1]) == onWordChar {
		endIdx++
	}

	// Convert rune indices back to display columns, accounting for border offset
	startCol := runeIndexToDisplayWidth(plainLine, startIdx) + borderOffset
	endCol := runeIndexToDisplayWidth(plainLine, endIdx+1) + borderOffset

	// Set selection
	m.selection.active = true
	m.selection.startLine = line
	m.selection.startCol = startCol
	m.selection.endLine = line
	m.selection.endCol = endCol
	m.selection.mouseButtonDown = false
}

// applySelectionHighlight applies selection highlighting to visible lines
func (m *model) applySelectionHighlight(lines []string, viewportStartLine int) []string {
	startLine, startCol, endLine, endCol := m.selection.normalized()

	highlighted := make([]string, len(lines))

	getLineWidth := func(line string) int {
		plainLine := ansi.Strip(line)
		trimmedLine := strings.TrimRight(plainLine, " \t")
		return runewidth.StringWidth(trimmedLine)
	}

	for i, line := range lines {
		absoluteLine := viewportStartLine + i

		if absoluteLine < startLine || absoluteLine > endLine {
			highlighted[i] = line
			continue
		}

		lineWidth := getLineWidth(line)
		switch {
		case startLine == endLine && absoluteLine == startLine:
			// Single line selection
			highlighted[i] = m.highlightLine(line, startCol, min(lineWidth, endCol))
		case absoluteLine == startLine:
			// Start of multi-line selection
			highlighted[i] = m.highlightLine(line, startCol, lineWidth)
		case absoluteLine == endLine:
			// End of multi-line selection
			highlighted[i] = m.highlightLine(line, 0, lineWidth)
		default:
			// Middle of multi-line selection
			highlighted[i] = m.highlightLine(line, 0, lineWidth)
		}
	}

	return highlighted
}

// highlightLine applies selection highlighting to a portion of a line
func (m *model) highlightLine(line string, startCol, endCol int) string {
	// Get plain text for boundary checks
	plainLine := ansi.Strip(line)
	plainWidth := runewidth.StringWidth(plainLine)

	// Validate and normalize boundaries
	if startCol >= plainWidth || startCol >= endCol {
		return line
	}
	endCol = min(endCol, plainWidth)

	// Extract the three parts while preserving ANSI codes
	before := ansi.Cut(line, 0, startCol)
	selectedText := ansi.Cut(line, startCol, endCol)
	selectedPlain := ansi.Strip(selectedText)
	selected := styles.SelectionStyle.Render(selectedPlain)
	after := ansi.Cut(line, endCol, plainWidth)

	return before + selected + after
}

// clearSelection resets the selection state
func (m *model) clearSelection() {
	m.selection.clear()
}
