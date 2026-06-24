package tui

import (
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/yankawai/kube-shipguard/internal/analyzer"
	"github.com/yankawai/kube-shipguard/internal/verdict"
)

type Model struct {
	findings   []analyzer.Finding
	filter     filterMode
	query      string
	searchMode bool
	showHelp   bool
	cursor     int
	width      int
	height     int
}

type filterMode string

const (
	filterAll    filterMode = "all"
	filterHigh   filterMode = "high"
	filterMedium filterMode = "medium"
	filterLow    filterMode = "low"
)

func New(findings []analyzer.Finding) Model {
	return Model{findings: findings, filter: filterAll}
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
		if m.searchMode {
			return m.updateSearch(msg)
		}
		switch msg.String() {
		case "q", "ctrl+c":
			return m, tea.Quit
		case "esc":
			m.showHelp = false
			m.searchMode = false
			m.query = ""
			m.cursor = 0
		case "?", "h":
			m.showHelp = !m.showHelp
		case "/":
			m.searchMode = true
		case "a":
			m.filter = filterAll
			m.cursor = 0
		case "1":
			m.filter = filterHigh
			m.cursor = 0
		case "2":
			m.filter = filterMedium
			m.cursor = 0
		case "3":
			m.filter = filterLow
			m.cursor = 0
		case "j", "down":
			if m.cursor < len(m.filteredFindings())-1 {
				m.cursor++
			}
		case "k", "up":
			if m.cursor > 0 {
				m.cursor--
			}
		case "home":
			m.cursor = 0
		case "end":
			if total := len(m.filteredFindings()); total > 0 {
				m.cursor = total - 1
			}
		}
	}
	return m.clampCursor(), nil
}

func (m Model) updateSearch(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "enter":
		m.searchMode = false
	case "esc":
		m.searchMode = false
		m.query = ""
		m.cursor = 0
	case "backspace", "ctrl+h":
		if len(m.query) > 0 {
			m.query = m.query[:len(m.query)-1]
			m.cursor = 0
		}
	default:
		if len(msg.Runes) > 0 {
			m.query += string(msg.Runes)
			m.cursor = 0
		}
	}
	return m.clampCursor(), nil
}

func (m Model) View() string {
	filtered := m.filteredFindings()
	if len(m.findings) == 0 {
		return frame(m.brandView(filtered)+"\n\n"+successStyle.Render("No findings. This manifest set is ready to ship."), m.width)
	}

	var builder strings.Builder
	builder.WriteString(m.brandView(filtered))
	builder.WriteString("\n\n")
	builder.WriteString(m.header(filtered))
	builder.WriteString("\n")
	builder.WriteString(m.toolbar())
	builder.WriteString("\n\n")
	if m.showHelp {
		builder.WriteString(m.helpView())
	} else if len(filtered) == 0 {
		builder.WriteString(emptyStyle.Render("No findings match the current filter. Press a for all or esc to clear search."))
	} else {
		builder.WriteString(m.listView(filtered))
		builder.WriteString("\n\n")
		builder.WriteString(m.detailView(filtered[m.cursor]))
	}
	builder.WriteString("\n\n")
	builder.WriteString(helpStyle.Render("j/k or up/down · 1 high · 2 medium · 3 low · a all · / search · ? help · q quit"))

	return frame(builder.String(), m.width)
}

func (m Model) header(filtered []analyzer.Finding) string {
	high, medium, low := countBySeverity(m.findings)
	filterLabel := string(m.filter)
	if m.query != "" {
		filterLabel += fmt.Sprintf(" · search=%q", m.query)
	}
	return fmt.Sprintf(
		"%s  %s  %s  %s  %s",
		titleStyle.Render(fmt.Sprintf("%d findings", len(m.findings))),
		highStyle.Render(fmt.Sprintf("%d high", high)),
		mediumStyle.Render(fmt.Sprintf("%d medium", medium)),
		lowStyle.Render(fmt.Sprintf("%d low", low)),
		mutedStyle.Render(fmt.Sprintf("showing %d · %s", len(filtered), filterLabel)),
	)
}

func (m Model) brandView(filtered []analyzer.Finding) string {
	high, medium, low := countBySeverity(m.findings)
	evaluation := verdict.Evaluate(m.findings)
	status := gateStatus(evaluation.Label)
	logo := logoStyle.Render(shipguardLogo)
	meta := strings.Join([]string{
		labelStyle.Render("KUBERNETES RELEASE READINESS"),
		fmt.Sprintf("%s  %s  %s", status.style.Render(status.label), mutedStyle.Render("static manifest review"), mutedStyle.Render("SARIF-ready")),
		fmt.Sprintf("%s  %s  %s  %s",
			highStyle.Render(fmt.Sprintf("%d high", high)),
			mediumStyle.Render(fmt.Sprintf("%d medium", medium)),
			lowStyle.Render(fmt.Sprintf("%d low", low)),
			mutedStyle.Render(fmt.Sprintf("%d visible", len(filtered))),
		),
		mutedStyle.Render(fmt.Sprintf("risk score %d", evaluation.RiskScore)),
		riskBar(high, medium, low),
	}, "\n")

	if m.width >= 132 {
		return lipgloss.JoinHorizontal(lipgloss.Top, logo, "  ", brandPanelStyle.Render(meta))
	}
	return logo + "\n" + brandPanelStyle.Render(meta)
}

type statusLabel struct {
	label string
	style lipgloss.Style
}

func gateStatus(label verdict.Label) statusLabel {
	switch label {
	case verdict.LabelBlock:
		return statusLabel{label: "GATE: BLOCK", style: highStyle}
	case verdict.LabelReview:
		return statusLabel{label: "GATE: REVIEW", style: mediumStyle}
	default:
		return statusLabel{label: "GATE: SHIP", style: successStyle}
	}
}

func riskBar(high, medium, low int) string {
	total := high + medium + low
	if total == 0 {
		return successStyle.Render("████████████████████") + mutedStyle.Render(" clean")
	}
	highWidth := scaledWidth(high, total)
	mediumWidth := scaledWidth(medium, total)
	lowWidth := scaledWidth(low, total)
	colored := highStyle.Render(strings.Repeat("█", highWidth))
	colored += mediumStyle.Render(strings.Repeat("█", mediumWidth))
	colored += lowStyle.Render(strings.Repeat("█", lowWidth))
	if fill := 20 - highWidth - mediumWidth - lowWidth; fill > 0 {
		colored += mutedStyle.Render(strings.Repeat("░", fill))
	}
	return colored + mutedStyle.Render(" risk mix")
}

func scaledWidth(value, total int) int {
	if value == 0 || total == 0 {
		return 0
	}
	width := value * 20 / total
	if width == 0 {
		return 1
	}
	return width
}

func (m Model) toolbar() string {
	search := "search: /"
	if m.searchMode {
		search = "search: " + activeSearchStyle.Render(m.query+"█")
	} else if m.query != "" {
		search = "search: " + activeSearchStyle.Render(m.query)
	}
	return subtlePanelStyle.Render("filters  [1] high  [2] medium  [3] low  [a] all") + "  " + subtlePanelStyle.Render(search)
}

func (m Model) listView(findings []analyzer.Finding) string {
	visible := 9
	if m.height > 32 {
		visible = 14
	}
	start := 0
	if m.cursor >= visible {
		start = m.cursor - visible + 1
	}
	end := start + visible
	if end > len(findings) {
		end = len(findings)
	}

	rows := make([]string, 0, end-start+1)
	rows = append(rows, tableHeaderStyle.Render("  SEV     RULE    RESOURCE                FILE                         MESSAGE"))
	for index := start; index < end; index++ {
		finding := findings[index]
		cursor := " "
		style := rowStyle
		if index == m.cursor {
			cursor = ">"
			style = activeRowStyle
		}
		severity := severityStyle(finding.Severity).Render(strings.ToUpper(string(finding.Severity)))
		resource := truncate(fmt.Sprintf("%s/%s", finding.Kind, finding.Name), 22)
		file := truncate(finding.File, 28)
		message := truncate(finding.Message, rowMessageWidth(m.width))
		rows = append(rows, style.Render(fmt.Sprintf("%s %-12s %-7s %-23s %-28s %s", cursor, severity, finding.RuleID, resource, file, message)))
	}
	return strings.Join(rows, "\n")
}

func (m Model) detailView(finding analyzer.Finding) string {
	rule := ruleCatalog[finding.RuleID]
	if rule == "" {
		rule = "Release-readiness guardrail."
	}
	detail := fmt.Sprintf(
		"%s\n%s\n\n%s\n\n%s\n%s\n\n%s",
		titleStyle.Render(finding.Location()),
		mutedStyle.Render(fmt.Sprintf("rule=%s severity=%s file=%s container=%s", finding.RuleID, finding.Severity, finding.File, defaultText(finding.Container, "-"))),
		finding.Message,
		accentStyle.Render("Why it matters: ")+rule,
		accentStyle.Render("Fix: ")+finding.Remediation,
		mutedStyle.Render(fmt.Sprintf("selected %d/%d", m.cursor+1, len(m.filteredFindings()))),
	)
	return panelStyle.Width(panelWidth(m.width)).Render(detail)
}

func (m Model) helpView() string {
	return panelStyle.Width(panelWidth(m.width)).Render(strings.Join([]string{
		titleStyle.Render("Interactive review mode"),
		"",
		"Navigation",
		"  j/down      next finding",
		"  k/up        previous finding",
		"  home/end    jump to first or last visible finding",
		"",
		"Filters",
		"  1           high findings only",
		"  2           medium findings only",
		"  3           low findings only",
		"  a           all findings",
		"",
		"Search",
		"  /           search rule id, kind, file, resource, message, remediation",
		"  esc         clear search or close help",
		"",
		"Exit",
		"  q           quit",
	}, "\n"))
}

func (m Model) filteredFindings() []analyzer.Finding {
	filtered := make([]analyzer.Finding, 0, len(m.findings))
	for _, finding := range m.findings {
		if m.filter != filterAll && filterMode(finding.Severity) != m.filter {
			continue
		}
		if m.query != "" && !matchesQuery(finding, m.query) {
			continue
		}
		filtered = append(filtered, finding)
	}
	return filtered
}

func (m Model) clampCursor() Model {
	total := len(m.filteredFindings())
	if total == 0 {
		m.cursor = 0
		return m
	}
	if m.cursor >= total {
		m.cursor = total - 1
	}
	if m.cursor < 0 {
		m.cursor = 0
	}
	return m
}

func matchesQuery(finding analyzer.Finding, query string) bool {
	haystack := strings.ToLower(strings.Join([]string{
		finding.RuleID,
		string(finding.Severity),
		finding.File,
		finding.Kind,
		finding.Namespace,
		finding.Name,
		finding.Container,
		finding.Message,
		finding.Remediation,
	}, " "))
	return strings.Contains(haystack, strings.ToLower(query))
}

func countBySeverity(findings []analyzer.Finding) (int, int, int) {
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
	return high, medium, low
}

func frame(body string, width int) string {
	contentWidth := width - 4
	if contentWidth < 96 {
		contentWidth = 96
	}
	return appStyle.Width(contentWidth).Render(body)
}

func truncate(value string, max int) string {
	if max < 4 || len(value) <= max {
		return value
	}
	return value[:max-1] + "…"
}

func rowMessageWidth(width int) int {
	if width < 130 {
		return 56
	}
	return width - 88
}

func panelWidth(width int) int {
	if width < 120 {
		return 92
	}
	return width - 10
}

func defaultText(value, fallback string) string {
	if value == "" {
		return fallback
	}
	return value
}

func severityStyle(severity analyzer.Severity) lipgloss.Style {
	switch severity {
	case analyzer.SeverityHigh:
		return highStyle
	case analyzer.SeverityMedium:
		return mediumStyle
	default:
		return lowStyle
	}
}

var ruleCatalog = map[string]string{
	"KSG001": "Readiness gates prevent traffic from reaching pods before they can serve requests.",
	"KSG002": "Liveness probes let Kubernetes restart stuck containers without manual intervention.",
	"KSG003": "Requests and limits make scheduling predictable and reduce noisy-neighbor risk.",
	"KSG004": "Privilege escalation can turn an application exploit into a broader container escape path.",
	"KSG005": "A read-only root filesystem limits persistence and tampering inside compromised containers.",
	"KSG006": "Mutable image tags make rollbacks and incident forensics unreliable.",
	"KSG007": "Running as root increases the blast radius of application and dependency vulnerabilities.",
	"KSG008": "Dropping Linux capabilities removes kernel privileges most workloads do not need.",
	"KSG009": "Single replicas turn node disruption or rolling updates into downtime.",
	"KSG010": "PodDisruptionBudgets protect availability during voluntary disruptions.",
	"KSG011": "NetworkPolicies reduce lateral movement and make workload traffic intent explicit.",
	"KSG012": "Committed Secret manifests often leak credentials through repository history.",
	"KSG013": "Sensitive ConfigMap keys usually mean secret material is bypassing secret-management controls.",
	"KSG014": "Direct LoadBalancer exposure should be intentional and reviewed.",
}

const shipguardLogo = `██╗  ██╗██╗   ██╗██████╗ ███████╗
██║ ██╔╝██║   ██║██╔══██╗██╔════╝
█████╔╝ ██║   ██║██████╔╝█████╗
██╔═██╗ ██║   ██║██╔══██╗██╔══╝
██║  ██╗╚██████╔╝██████╔╝███████╗
╚═╝  ╚═╝ ╚═════╝ ╚═════╝ ╚══════╝
        SHIPGUARD`

var (
	appStyle          = lipgloss.NewStyle().Padding(1, 2)
	logoStyle         = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("14"))
	labelStyle        = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("15"))
	titleStyle        = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("15"))
	mutedStyle        = lipgloss.NewStyle().Foreground(lipgloss.Color("8"))
	helpStyle         = lipgloss.NewStyle().Foreground(lipgloss.Color("8"))
	emptyStyle        = lipgloss.NewStyle().Foreground(lipgloss.Color("14")).Bold(true)
	successStyle      = lipgloss.NewStyle().Foreground(lipgloss.Color("10")).Bold(true)
	accentStyle       = lipgloss.NewStyle().Foreground(lipgloss.Color("14")).Bold(true)
	highStyle         = lipgloss.NewStyle().Foreground(lipgloss.Color("9")).Bold(true)
	mediumStyle       = lipgloss.NewStyle().Foreground(lipgloss.Color("11")).Bold(true)
	lowStyle          = lipgloss.NewStyle().Foreground(lipgloss.Color("12")).Bold(true)
	tableHeaderStyle  = lipgloss.NewStyle().Foreground(lipgloss.Color("8")).Bold(true)
	rowStyle          = lipgloss.NewStyle().Foreground(lipgloss.Color("7"))
	activeRowStyle    = lipgloss.NewStyle().Foreground(lipgloss.Color("15")).Bold(true)
	panelStyle        = lipgloss.NewStyle().Border(lipgloss.RoundedBorder()).BorderForeground(lipgloss.Color("8")).Padding(1, 2)
	brandPanelStyle   = lipgloss.NewStyle().Border(lipgloss.NormalBorder()).BorderForeground(lipgloss.Color("8")).Padding(1, 2)
	subtlePanelStyle  = lipgloss.NewStyle().Foreground(lipgloss.Color("7")).Background(lipgloss.Color("0")).Padding(0, 1)
	activeSearchStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("14")).Bold(true)
)
