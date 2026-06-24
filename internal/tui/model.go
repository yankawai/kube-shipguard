package tui

import (
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/yankawai/kube-shipguard/internal/analyzer"
)

type Model struct {
	findings []analyzer.Finding
	cursor   int
	width    int
	height   int
}

func New(findings []analyzer.Finding) Model {
	return Model{findings: findings}
}

func (m Model) Init() tea.Cmd {
	return nil
}

func (m Model) Update(message tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := message.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
	case tea.KeyMsg:
		switch msg.String() {
		case "q", "ctrl+c", "esc":
			return m, tea.Quit
		case "j", "down":
			if m.cursor < len(m.findings)-1 {
				m.cursor++
			}
		case "k", "up":
			if m.cursor > 0 {
				m.cursor--
			}
		case "home":
			m.cursor = 0
		case "end":
			if len(m.findings) > 0 {
				m.cursor = len(m.findings) - 1
			}
		}
	}
	return m, nil
}

func (m Model) View() string {
	if len(m.findings) == 0 {
		return frame("Kube ShipGuard", successStyle.Render("No findings. This manifest set is ready to ship."), m.width)
	}

	var builder strings.Builder
	builder.WriteString(header(m.findings))
	builder.WriteString("\n\n")
	builder.WriteString(m.listView())
	builder.WriteString("\n\n")
	builder.WriteString(m.detailView())
	builder.WriteString("\n\n")
	builder.WriteString(helpStyle.Render("up/down or k/j to navigate · home/end · q to quit"))

	return frame("Kube ShipGuard", builder.String(), m.width)
}

func (m Model) listView() string {
	visible := 10
	if m.height > 24 {
		visible = 14
	}
	start := 0
	if m.cursor >= visible {
		start = m.cursor - visible + 1
	}
	end := start + visible
	if end > len(m.findings) {
		end = len(m.findings)
	}

	rows := make([]string, 0, end-start)
	for index := start; index < end; index++ {
		finding := m.findings[index]
		cursor := " "
		style := rowStyle
		if index == m.cursor {
			cursor = ">"
			style = activeRowStyle
		}
		rows = append(rows, style.Render(fmt.Sprintf("%s %-6s %-6s %s %s", cursor, strings.ToUpper(string(finding.Severity)), finding.RuleID, finding.File, finding.Message)))
	}
	return strings.Join(rows, "\n")
}

func (m Model) detailView() string {
	finding := m.findings[m.cursor]
	detail := fmt.Sprintf(
		"%s\n%s\n\n%s\n%s",
		titleStyle.Render(finding.Location()),
		mutedStyle.Render(fmt.Sprintf("kind=%s namespace=%s name=%s container=%s", finding.Kind, finding.Namespace, finding.Name, defaultText(finding.Container, "-"))),
		finding.Message,
		accentStyle.Render("Fix: ")+finding.Remediation,
	)
	return panelStyle.Render(detail)
}

func header(findings []analyzer.Finding) string {
	var high, medium, low int
	for _, finding := range findings {
		switch finding.Severity {
		case analyzer.SeverityHigh:
			high++
		case analyzer.SeverityMedium:
			medium++
		case analyzer.SeverityLow:
			low++
		}
	}
	return fmt.Sprintf(
		"%s  %s  %s  %s",
		titleStyle.Render(fmt.Sprintf("%d findings", len(findings))),
		highStyle.Render(fmt.Sprintf("%d high", high)),
		mediumStyle.Render(fmt.Sprintf("%d medium", medium)),
		lowStyle.Render(fmt.Sprintf("%d low", low)),
	)
}

func frame(title, body string, width int) string {
	contentWidth := width - 4
	if contentWidth < 80 {
		contentWidth = 80
	}
	return appStyle.Width(contentWidth).Render(titleStyle.Render(title) + "\n\n" + body)
}

func defaultText(value, fallback string) string {
	if value == "" {
		return fallback
	}
	return value
}

var (
	appStyle       = lipgloss.NewStyle().Padding(1, 2)
	titleStyle     = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("15"))
	mutedStyle     = lipgloss.NewStyle().Foreground(lipgloss.Color("8"))
	helpStyle      = lipgloss.NewStyle().Foreground(lipgloss.Color("8"))
	successStyle   = lipgloss.NewStyle().Foreground(lipgloss.Color("10")).Bold(true)
	accentStyle    = lipgloss.NewStyle().Foreground(lipgloss.Color("14")).Bold(true)
	highStyle      = lipgloss.NewStyle().Foreground(lipgloss.Color("9")).Bold(true)
	mediumStyle    = lipgloss.NewStyle().Foreground(lipgloss.Color("11")).Bold(true)
	lowStyle       = lipgloss.NewStyle().Foreground(lipgloss.Color("12")).Bold(true)
	rowStyle       = lipgloss.NewStyle().Foreground(lipgloss.Color("7"))
	activeRowStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("15")).Bold(true)
	panelStyle     = lipgloss.NewStyle().Border(lipgloss.RoundedBorder()).BorderForeground(lipgloss.Color("8")).Padding(1, 2)
)
