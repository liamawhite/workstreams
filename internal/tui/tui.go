package tui

import (
	"fmt"
	"sort"
	"strings"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/liamawhite/workstreams/pkg/types"
	"github.com/liamawhite/workstreams/pkg/workstreams"
)

const (
	boxContentWidth  = 44
	boxGap           = 4
	dialogInnerWidth = 44
)

type mode int

const (
	modeCarousel mode = iota
	modeAdd
	modeConfirmDelete
)

// Result is returned by Run to tell the caller what action was requested.
type Result struct {
	// Selected is the workstream dir name to switch to (set on Enter or after create).
	Selected string
	// CreateName/CreateType are set when the user confirmed a new workstream.
	// The caller is responsible for running Create so output streams to the terminal.
	CreateName string
	CreateType string
}

var (
	activeBoxStyle = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color("205")).
			Width(boxContentWidth)

	inactiveBoxStyle = lipgloss.NewStyle().
				Border(lipgloss.RoundedBorder()).
				BorderForeground(lipgloss.Color("240")).
				Width(boxContentWidth)

	activeTitleStyle   = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("205"))
	inactiveTitleStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("240"))
	cursorStyle        = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("205"))
	dimItemStyle       = lipgloss.NewStyle().Foreground(lipgloss.Color("240"))
	helpStyle          = lipgloss.NewStyle().Foreground(lipgloss.Color("240"))
	errorStyle         = lipgloss.NewStyle().Foreground(lipgloss.Color("9"))
	warningStyle       = lipgloss.NewStyle().Foreground(lipgloss.Color("214")).Bold(true)

	dialogStyle = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color("205")).
			Padding(1, 2).
			Width(dialogInnerWidth)

	deleteDialogStyle = lipgloss.NewStyle().
				Border(lipgloss.RoundedBorder()).
				BorderForeground(lipgloss.Color("9")).
				Padding(1, 2).
				Width(dialogInnerWidth)
)

type model struct {
	typeKeys []string
	byType   map[string][]string
	labels   map[string]string
	typeIdx  int
	cursor   int
	result   Result
	width    int
	height   int

	mode     mode
	addInput textinput.Model
	addErr   string
}

func newModel() (model, error) {
	typeList, err := types.List()
	if err != nil {
		return model{}, fmt.Errorf("loading types: %w", err)
	}

	byType, err := workstreams.ListByType()
	if err != nil {
		return model{}, fmt.Errorf("loading workstreams: %w", err)
	}

	labels := make(map[string]string)
	for _, t := range typeList {
		cfg, err := types.ReadConfig(t)
		if err != nil || cfg == nil {
			labels[t] = t
		} else {
			labels[t] = cfg.Name
		}
	}
	labels[""] = "(no type)"

	known := make(map[string]bool)
	for _, t := range typeList {
		known[t] = true
	}
	for key := range byType {
		if key != "" {
			known[key] = true
			if _, ok := labels[key]; !ok {
				labels[key] = key
			}
		}
	}

	allTypes := make([]string, 0, len(known))
	for t := range known {
		allTypes = append(allTypes, t)
	}
	sort.Slice(allTypes, func(i, j int) bool {
		ci, cj := len(byType[allTypes[i]]), len(byType[allTypes[j]])
		if ci != cj {
			return ci > cj
		}
		return allTypes[i] < allTypes[j]
	})

	typeKeys := append([]string{}, allTypes...)
	if len(byType[""]) > 0 {
		typeKeys = append(typeKeys, "")
	}

	ti := textinput.New()
	ti.Placeholder = "workstream name"
	ti.CharLimit = 64
	ti.Width = dialogInnerWidth

	return model{
		typeKeys: typeKeys,
		byType:   byType,
		labels:   labels,
		width:    80,
		height:   24,
		addInput: ti,
	}, nil
}

func (m model) Init() tea.Cmd { return nil }

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	if size, ok := msg.(tea.WindowSizeMsg); ok {
		m.width = size.Width
		m.height = size.Height
		return m, nil
	}
	switch m.mode {
	case modeAdd:
		return m.updateAdd(msg)
	case modeConfirmDelete:
		return m.updateConfirmDelete(msg)
	default:
		return m.updateCarousel(msg)
	}
}

func (m model) updateCarousel(msg tea.Msg) (tea.Model, tea.Cmd) {
	key, ok := msg.(tea.KeyMsg)
	if !ok {
		return m, nil
	}
	switch key.String() {
	case "q", "ctrl+c":
		return m, tea.Quit
	case "h", "left":
		if len(m.typeKeys) > 0 {
			m.typeIdx = (m.typeIdx - 1 + len(m.typeKeys)) % len(m.typeKeys)
			m.cursor = 0
		}
	case "l", "right":
		if len(m.typeKeys) > 0 {
			m.typeIdx = (m.typeIdx + 1) % len(m.typeKeys)
			m.cursor = 0
		}
	case "j", "down":
		if len(m.typeKeys) > 0 {
			items := m.byType[m.typeKeys[m.typeIdx]]
			if m.cursor < len(items)-1 {
				m.cursor++
			}
		}
	case "k", "up":
		if m.cursor > 0 {
			m.cursor--
		}
	case "enter":
		if len(m.typeKeys) > 0 {
			items := m.byType[m.typeKeys[m.typeIdx]]
			if m.cursor < len(items) {
				m.result.Selected = items[m.cursor]
				return m, tea.Quit
			}
		}
	case "a":
		if len(m.typeKeys) > 0 {
			m.addInput.SetValue("")
			m.addInput.Focus()
			m.addErr = ""
			m.mode = modeAdd
		}
	case "d":
		if len(m.typeKeys) > 0 {
			items := m.byType[m.typeKeys[m.typeIdx]]
			if len(items) > 0 && m.cursor < len(items) {
				m.mode = modeConfirmDelete
			}
		}
	}
	return m, nil
}

func (m model) updateAdd(msg tea.Msg) (tea.Model, tea.Cmd) {
	key, ok := msg.(tea.KeyMsg)
	if !ok {
		var cmd tea.Cmd
		m.addInput, cmd = m.addInput.Update(msg)
		return m, cmd
	}
	switch key.String() {
	case "ctrl+c":
		return m, tea.Quit
	case "esc":
		m.mode = modeCarousel
		m.addErr = ""
		return m, nil
	case "enter":
		name := strings.TrimSpace(m.addInput.Value())
		if name == "" {
			m.addErr = "name cannot be empty"
			return m, nil
		}
		if ok, _ := workstreams.Exists(workstreams.ToDirName(name)); ok {
			m.addErr = fmt.Sprintf("%q already exists", name)
			return m, nil
		}
		m.result.CreateName = name
		m.result.CreateType = m.typeKeys[m.typeIdx]
		return m, tea.Quit
	}
	var cmd tea.Cmd
	m.addInput, cmd = m.addInput.Update(msg)
	return m, cmd
}

func (m model) updateConfirmDelete(msg tea.Msg) (tea.Model, tea.Cmd) {
	key, ok := msg.(tea.KeyMsg)
	if !ok {
		return m, nil
	}
	switch key.String() {
	case "ctrl+c":
		return m, tea.Quit
	case "esc", "n", "N":
		m.mode = modeCarousel
		return m, nil
	case "y", "Y", "enter":
		items := m.byType[m.typeKeys[m.typeIdx]]
		if len(items) > 0 && m.cursor < len(items) {
			name := items[m.cursor]
			workstreams.DeleteAsync(name) //nolint:errcheck

			// Optimistically remove from in-memory state so the UI updates immediately.
			filtered := make([]string, 0, len(items)-1)
			for _, item := range items {
				if item != name {
					filtered = append(filtered, item)
				}
			}
			m.byType[m.typeKeys[m.typeIdx]] = filtered

			sort.Slice(m.typeKeys, func(i, j int) bool {
				ci, cj := len(m.byType[m.typeKeys[i]]), len(m.byType[m.typeKeys[j]])
				if ci != cj {
					return ci > cj
				}
				return m.typeKeys[i] < m.typeKeys[j]
			})
			remaining := m.byType[m.typeKeys[m.typeIdx]]
			if m.cursor >= len(remaining) {
				m.cursor = len(remaining) - 1
			}
			if m.cursor < 0 {
				m.cursor = 0
			}
		}
		m.mode = modeCarousel
		return m, nil
	}
	return m, nil
}

func (m model) visibleCount() int {
	boxTotal := boxContentWidth + 2
	if m.width <= 0 || boxTotal <= 0 {
		return 1
	}
	n := (m.width + boxGap) / (boxTotal + boxGap)
	if n < 1 {
		n = 1
	}
	if n > len(m.typeKeys) {
		n = len(m.typeKeys)
	}
	return n
}

func (m model) viewStart() int {
	visible := m.visibleCount()
	n := len(m.typeKeys)
	if visible >= n {
		return 0
	}
	start := m.typeIdx - visible/2
	if start < 0 {
		start = 0
	}
	if start+visible > n {
		start = n - visible
	}
	return start
}

func (m model) listHeight() int {
	max := 1
	for _, items := range m.byType {
		if len(items) > max {
			max = len(items)
		}
	}
	cap := m.height - 7
	if cap < 1 {
		cap = 1
	}
	if max > cap {
		return cap
	}
	return max
}

func truncate(s string, max int) string {
	r := []rune(s)
	if len(r) <= max {
		return s
	}
	return string(r[:max-1]) + "…"
}

func (m model) renderBox(key string, active bool) string {
	items := m.byType[key]
	h := m.listHeight()

	title := fmt.Sprintf("%s (%d)", m.labels[key], len(items))
	title = truncate(title, boxContentWidth)
	if active {
		title = activeTitleStyle.Render(title)
	} else {
		title = inactiveTitleStyle.Render(title)
	}

	divider := strings.Repeat("─", boxContentWidth)

	offset := 0
	if active && m.cursor >= h {
		offset = m.cursor - h + 1
	}

	rows := make([]string, h)
	for i := range rows {
		idx := i + offset
		if idx >= len(items) {
			rows[i] = ""
			continue
		}
		name := truncate(items[idx], boxContentWidth-2)
		switch {
		case active && idx == m.cursor:
			rows[i] = cursorStyle.Render("> " + name)
		case active:
			rows[i] = "  " + name
		default:
			rows[i] = dimItemStyle.Render("  " + name)
		}
	}

	content := title + "\n" + divider + "\n" + strings.Join(rows, "\n")
	if active {
		return activeBoxStyle.Render(content)
	}
	return inactiveBoxStyle.Render(content)
}

func (m model) viewCarousel() string {
	help := lipgloss.Place(m.width, 1, lipgloss.Center, lipgloss.Center,
		helpStyle.Render("h/← l/→ type  j/k navigate  enter switch  a add  d delete  q quit"))

	bodyHeight := m.height - 1

	if len(m.typeKeys) == 0 {
		msg := "No workstream types defined.\n\n" +
			helpStyle.Render("workstreams types add <name>")
		body := lipgloss.Place(m.width, bodyHeight, lipgloss.Center, lipgloss.Center, msg)
		return body + "\n" + help
	}

	visible := m.visibleCount()
	start := m.viewStart()
	end := start + visible
	if end > len(m.typeKeys) {
		end = len(m.typeKeys)
	}

	boxes := make([]string, end-start)
	for i, key := range m.typeKeys[start:end] {
		boxes[i] = m.renderBox(key, start+i == m.typeIdx)
	}

	gap := strings.Repeat(" ", boxGap)
	parts := make([]string, len(boxes)*2-1)
	for i, b := range boxes {
		parts[i*2] = b
		if i < len(boxes)-1 {
			parts[i*2+1] = gap
		}
	}
	row := lipgloss.JoinHorizontal(lipgloss.Top, parts...)

	var indicators strings.Builder
	if start > 0 {
		indicators.WriteString(helpStyle.Render("←  "))
	}
	if end < len(m.typeKeys) {
		indicators.WriteString(helpStyle.Render("  →"))
	}

	var inner strings.Builder
	inner.WriteString(row)
	if indicators.Len() > 0 {
		inner.WriteString("\n")
		inner.WriteString(indicators.String())
	}

	body := lipgloss.Place(m.width, bodyHeight, lipgloss.Center, lipgloss.Center, inner.String())
	return body + "\n" + help
}

func (m model) viewAdd() string {
	typeLabel := m.labels[m.typeKeys[m.typeIdx]]

	var content strings.Builder
	content.WriteString(activeTitleStyle.Render("New workstream") + "\n")
	content.WriteString(dimItemStyle.Render("Type: "+typeLabel) + "\n\n")
	content.WriteString(m.addInput.View() + "\n\n")
	if m.addErr != "" {
		content.WriteString(errorStyle.Render(m.addErr) + "\n\n")
	}
	content.WriteString(helpStyle.Render("enter confirm  esc cancel"))

	return lipgloss.Place(m.width, m.height, lipgloss.Center, lipgloss.Center,
		dialogStyle.Render(content.String()))
}

func (m model) viewConfirmDelete() string {
	items := m.byType[m.typeKeys[m.typeIdx]]
	name := ""
	if len(items) > 0 && m.cursor < len(items) {
		name = items[m.cursor]
	}

	var content strings.Builder
	content.WriteString(warningStyle.Render("Delete workstream?") + "\n\n")
	content.WriteString(fmt.Sprintf("  %s\n\n", name))
	content.WriteString(errorStyle.Render("This cannot be undone.") + "\n\n")
	content.WriteString(helpStyle.Render("y/enter confirm  n/esc cancel"))

	return lipgloss.Place(m.width, m.height, lipgloss.Center, lipgloss.Center,
		deleteDialogStyle.Render(content.String()))
}

func (m model) View() string {
	switch m.mode {
	case modeAdd:
		return m.viewAdd()
	case modeConfirmDelete:
		return m.viewConfirmDelete()
	default:
		return m.viewCarousel()
	}
}

// Run launches the TUI and returns a Result describing what the user requested.
func Run() (Result, error) {
	m, err := newModel()
	if err != nil {
		return Result{}, err
	}
	out, err := tea.NewProgram(m, tea.WithAltScreen()).Run()
	if err != nil {
		return Result{}, err
	}
	if final, ok := out.(model); ok {
		return final.result, nil
	}
	return Result{}, nil
}
