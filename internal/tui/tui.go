package tui

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"time"

	"dossier/internal/core"

	"github.com/charmbracelet/bubbles/table"
	"github.com/charmbracelet/bubbles/textinput"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/glamour"
	"github.com/charmbracelet/glamour/styles"
	"github.com/charmbracelet/lipgloss"
	"github.com/fsnotify/fsnotify"
)

// View represents the current active TUI screen view.
type View int

const (
	ViewDashboard View = iota
	ViewDetail
	ViewStatusPicker
	ViewNextActionEditor
	ViewLeadEditor
	ViewPriorityEditor
	ViewLinkInput
	ViewLinkSelector
	ViewMergeSelector
	ViewMergeConflictResolver
)

// Styling tokens
var (
	purple       = lipgloss.Color("99")
	lightGray    = lipgloss.Color("7") // Use terminal's standard light gray (ANSI 7)
	darkGray     = lipgloss.Color("8") // Use terminal's standard dark gray/bright black (ANSI 8)
	vibrantGreen = lipgloss.Color("2") // Use terminal's standard green (ANSI 2)
	vibrantRed   = lipgloss.Color("1") // Use terminal's standard red (ANSI 1)
	warningGold  = lipgloss.Color("3") // Use terminal's standard yellow/gold (ANSI 3)

	titleStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("15")). // Use terminal's standard bright white (ANSI 15) for title text on purple bg
			Background(purple).
			Padding(0, 2).
			Bold(true)

	subtitleStyle = lipgloss.NewStyle().
			Foreground(darkGray). // Inherit terminal theme's gray (ANSI 8)
			Italic(true)

	headerStyle = lipgloss.NewStyle().
			Foreground(purple).
			Bold(true)

	footerStyle = lipgloss.NewStyle().
			Reverse(true). // Inverted foreground and background dynamically to match terminal theme status bar
			Padding(0, 1)

	warningStyle = lipgloss.NewStyle().
			Foreground(warningGold).
			Bold(true)

	errorStyle = lipgloss.NewStyle().
			Foreground(vibrantRed).
			Bold(true)

	metaLabelStyle = lipgloss.NewStyle().
			Foreground(purple).
			Bold(true)

	metaValueStyle = lipgloss.NewStyle() // Inherit terminal's default text foreground color

	statusActiveStyle   = lipgloss.NewStyle().Foreground(vibrantGreen).Bold(true)
	statusWaitingStyle  = lipgloss.NewStyle().Foreground(warningGold)
	statusBlockedStyle  = lipgloss.NewStyle().Foreground(vibrantRed).Bold(true)
	statusResolvedStyle = lipgloss.NewStyle().Foreground(darkGray)
	statusArchivedStyle = lipgloss.NewStyle().Foreground(darkGray).Faint(true)

	editorBoxStyle = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(purple).
			Padding(1, 2).
			Margin(1, 0)

	focusedItemStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color("15")). // Use terminal's standard bright white (ANSI 15)
				Background(purple).
				Bold(true).
				Padding(0, 1)

	activeOptionStyle = lipgloss.NewStyle().
				Foreground(vibrantGreen).
				Bold(true)
)

// Messages
type listDossiersMsg []core.ListItem
type recallDossierMsg struct {
	id       string
	result   core.RecallResult
	err      error
	warnings []core.Warning
}
type mutationResultMsg struct {
	err      error
	prevView View
	targetID string
}
type linkResultMsg struct {
	err     error
	result  core.Result
	content string
}
type linkConfirmResultMsg struct {
	err error
}
type mergeResultMsg struct {
	err      error
	result   core.Result
	sourceID string
	targetID string
}
type errMsg error

type dossierUpdatedMsg struct{}

func waitForUpdate(updateChan <-chan string) tea.Cmd {
	return func() tea.Msg {
		<-updateChan
		return dossierUpdatedMsg{}
	}
}

type targetDossier struct {
	id           string
	name         string
	status       core.Status
	importance   core.Importance
	urgency      core.Urgency
	dueDate      string
	nextAction   string
	lead         string
	baseRevision core.Revision
}

// Model holds the application state.
type Model struct {
	svc         *core.Service
	currentView View

	// Data
	items        []core.ListItem
	recallResult core.RecallResult

	// Viewport & Table
	table            table.Model
	viewport         viewport.Model
	conflictViewport viewport.Model
	width            int
	height           int

	// Error / Warning tracking
	err      error
	warnings []core.Warning

	// View state helpers
	loading bool

	// Mutation target cache
	previousView       View
	targetID           string
	targetName         string
	targetBaseRevision core.Revision

	// Status Picker view state
	statusOptions []core.Status
	statusCursor  int

	// Next Action Editor view state
	nextActionInput textinput.Model

	// Lead Editor view state
	leadInput textinput.Model

	// Priority Editor view state
	priorityFocus  int // 0 = Importance, 1 = Urgency, 2 = Due Date
	editImportance core.Importance
	editUrgency    core.Urgency
	dueDateInput   textinput.Model

	// Link view state
	linkTextInput   textinput.Model
	linkContent     string
	linkSuggestions []core.Suggestion
	linkCursor      int

	// Merge view state
	mergeSourceID          string
	mergeSourceName        string
	mergeTargetID          string
	mergeTargets           []core.ListItem
	mergeCursor            int
	mergeConflict          *core.Conflict
	conflictResolverCursor int // 0 = Resolve/Force, 1 = Cancel

	watcher      *fsnotify.Watcher
	updateChan   chan string
	watchedPaths map[string]bool

	// Cached markdown renderer, rebuilt only when the wrap width changes.
	mdRenderer      *glamour.TermRenderer
	mdRendererWidth int
}

// NewModel instantiates the root TUI model.
func NewModel(svc *core.Service) Model {
	// Initialize default empty table
	columns := []table.Column{
		{Title: "Name", Width: 18},
		{Title: "Status", Width: 8},
		{Title: "Lead", Width: 8},
		{Title: "Priority", Width: 12},
		{Title: "Next Action", Width: 13},
		{Title: "Due", Width: 8},
	}

	t := table.New(
		table.WithColumns(columns),
		table.WithFocused(true),
	)

	s := table.DefaultStyles()
	s.Header = s.Header.
		BorderStyle(lipgloss.NormalBorder()).
		BorderForeground(darkGray).
		BorderBottom(true).
		Bold(true)
	s.Selected = s.Selected.
		Foreground(lipgloss.Color("15")).
		Background(purple).
		Bold(true)
	t.SetStyles(s)

	vp := viewport.New(0, 0)
	cvp := viewport.New(0, 0)

	statusOptions := []core.Status{
		core.StatusActive,
		core.StatusWaiting,
		core.StatusBlocked,
		core.StatusResolved,
		core.StatusArchived,
	}

	watcher, err := fsnotify.NewWatcher()
	updateChan := make(chan string, 100)
	if err == nil {
		go func() {
			for {
				select {
				case event, ok := <-watcher.Events:
					if !ok {
						return
					}
					if event.Op.Has(fsnotify.Write) || event.Op.Has(fsnotify.Rename) || event.Op.Has(fsnotify.Create) {
						updateChan <- "update"
					}
				case <-watcher.Errors:
				}
			}
		}()
	}

	return Model{
		svc:              svc,
		currentView:      ViewDashboard,
		table:            t,
		viewport:         vp,
		conflictViewport: cvp,
		loading:          true,
		statusOptions:    statusOptions,
		watcher:          watcher,
		updateChan:       updateChan,
		watchedPaths:     map[string]bool{},
	}
}

// syncWatches makes the fsnotify watch set exactly match paths, adding new ones
// and dropping stale ones. Failures to add/remove a single path are non-fatal.
func (m *Model) syncWatches(paths []string) {
	if m.watcher == nil {
		return
	}
	desired := make(map[string]bool, len(paths))
	for _, p := range paths {
		if p == "" {
			continue
		}
		desired[p] = true
		if !m.watchedPaths[p] {
			if err := m.watcher.Add(p); err == nil {
				m.watchedPaths[p] = true
			}
		}
	}
	for p := range m.watchedPaths {
		if !desired[p] {
			_ = m.watcher.Remove(p)
			delete(m.watchedPaths, p)
		}
	}
}

// ensureWatch adds a single path to the watch set without disturbing the others.
func (m *Model) ensureWatch(path string) {
	if m.watcher == nil || path == "" || m.watchedPaths[path] {
		return
	}
	if err := m.watcher.Add(path); err == nil {
		m.watchedPaths[path] = true
	}
}

// Init initializes the tea program, triggering initial loads.
func (m Model) Init() tea.Cmd {
	return tea.Batch(m.listDossiersCmd(), waitForUpdate(m.updateChan))
}

// listDossiersCmd fetches the dossier list asynchronously.
func (m Model) listDossiersCmd() tea.Cmd {
	return func() tea.Msg {
		// Use empty status to query active work (active/waiting/blocked) by default as planned
		res, err := m.svc.List(context.Background(), core.ListReq{Status: ""})
		if err != nil {
			return errMsg(err)
		}
		items, ok := res.Data.([]core.ListItem)
		if !ok {
			return errMsg(fmt.Errorf("invalid list data type"))
		}
		return listDossiersMsg(items)
	}
}

// recallDossierCmd fetches the details of a specific dossier.
func (m Model) recallDossierCmd(id string) tea.Cmd {
	return func() tea.Msg {
		res, err := m.svc.Recall(context.Background(), core.RecallReq{ID: id})
		if err != nil {
			return recallDossierMsg{id: id, err: err}
		}
		recallRes, ok := res.Data.(core.RecallResult)
		if !ok {
			return recallDossierMsg{id: id, err: fmt.Errorf("invalid recall data type")}
		}
		return recallDossierMsg{
			id:       id,
			result:   recallRes,
			warnings: res.Warnings,
		}
	}
}

func (m Model) firstLinkCmd(content string) tea.Cmd {
	return func() tea.Msg {
		res, err := m.svc.Link(context.Background(), core.LinkReq{
			ID:      "",
			Content: content,
			Title:   "TUI Interactive Link",
		})
		return linkResultMsg{err: err, result: res, content: content}
	}
}

func (m Model) confirmLinkCmd(id string, content string) tea.Cmd {
	return func() tea.Msg {
		_, err := m.svc.Link(context.Background(), core.LinkReq{
			ID:      id,
			Content: content,
			Title:   "TUI Interactive Link",
		})
		return linkConfirmResultMsg{err: err}
	}
}

func (m Model) mergeCmd(sourceID, targetID string, resolved []string) tea.Cmd {
	return func() tea.Msg {
		res, err := m.svc.Merge(context.Background(), core.MergeReq{
			SourceID:          sourceID,
			TargetID:          targetID,
			ResolvedConflicts: resolved,
		})
		return mergeResultMsg{
			err:      err,
			result:   res,
			sourceID: sourceID,
			targetID: targetID,
		}
	}
}

func (m Model) setStatusCmd(id string, baseRev core.Revision, status core.Status) tea.Cmd {
	return func() tea.Msg {
		_, err := m.svc.Save(context.Background(), core.SaveReq{
			ID:                 id,
			BaseRevision:       baseRev,
			FrontmatterUpdates: map[string]any{"status": string(status)},
		})
		return mutationResultMsg{err: err, prevView: m.previousView, targetID: id}
	}
}

func (m Model) saveNextActionCmd(id string, baseRev core.Revision, nextAction string) tea.Cmd {
	return func() tea.Msg {
		_, err := m.svc.Save(context.Background(), core.SaveReq{
			ID:                 id,
			BaseRevision:       baseRev,
			FrontmatterUpdates: map[string]any{"next_action": nextAction},
		})
		return mutationResultMsg{err: err, prevView: m.previousView, targetID: id}
	}
}

func (m Model) saveLeadCmd(id string, baseRev core.Revision, lead string) tea.Cmd {
	return func() tea.Msg {
		_, err := m.svc.Save(context.Background(), core.SaveReq{
			ID:                 id,
			BaseRevision:       baseRev,
			FrontmatterUpdates: map[string]any{"lead": lead},
		})
		return mutationResultMsg{err: err, prevView: m.previousView, targetID: id}
	}
}

func (m Model) savePriorityCmd(id string, baseRev core.Revision, importance core.Importance, urgency core.Urgency, dueDate string) tea.Cmd {
	return func() tea.Msg {
		_, err := m.svc.Save(context.Background(), core.SaveReq{
			ID:           id,
			BaseRevision: baseRev,
			FrontmatterUpdates: map[string]any{
				"importance": string(importance),
				"urgency":    string(urgency),
				"due_date":   dueDate,
			},
		})
		return mutationResultMsg{err: err, prevView: m.previousView, targetID: id}
	}
}

func (m Model) getTargetDossier() (targetDossier, bool) {
	if m.currentView == ViewDetail {
		fm := m.recallResult.Frontmatter
		return targetDossier{
			id:           fm.ID,
			name:         fm.Name,
			status:       fm.Status,
			importance:   fm.Importance,
			urgency:      fm.Urgency,
			dueDate:      fm.DueDate,
			nextAction:   fm.NextAction,
			lead:         fm.Lead,
			baseRevision: m.recallResult.Revision,
		}, true
	}

	// Dashboard view
	idx := m.table.Cursor()
	if idx >= 0 && idx < len(m.items) {
		item := m.items[idx]
		return targetDossier{
			id:           item.ID,
			name:         item.Name,
			status:       core.Status(item.Status),
			importance:   core.Importance(item.Importance),
			urgency:      core.Urgency(item.Urgency),
			dueDate:      item.DueDate,
			nextAction:   item.NextAction,
			lead:         item.Lead,
			baseRevision: "", // Skip check from dashboard
		}, true
	}
	return targetDossier{}, false
}

func (m *Model) startEditStatus(t targetDossier) {
	m.previousView = m.currentView
	m.currentView = ViewStatusPicker
	m.targetID = t.id
	m.targetName = t.name

	m.statusCursor = 0
	for i, o := range m.statusOptions {
		if o == t.status {
			m.statusCursor = i
			break
		}
	}
}

func (m *Model) startEditNextAction(t targetDossier) {
	m.previousView = m.currentView
	m.currentView = ViewNextActionEditor
	m.targetID = t.id
	m.targetName = t.name
	m.targetBaseRevision = t.baseRevision

	m.nextActionInput = textinput.New()
	m.nextActionInput.SetValue(t.nextAction)
	m.nextActionInput.Focus()
	m.nextActionInput.Width = 60
}

func (m *Model) startEditLead(t targetDossier) {
	m.previousView = m.currentView
	m.currentView = ViewLeadEditor
	m.targetID = t.id
	m.targetName = t.name

	m.leadInput = textinput.New()
	m.leadInput.Placeholder = "e.g. Alice"
	m.leadInput.SetValue(t.lead)
	m.leadInput.Focus()
	m.leadInput.Width = 40
}

func (m *Model) startEditPriority(t targetDossier) {
	m.previousView = m.currentView
	m.currentView = ViewPriorityEditor
	m.targetID = t.id
	m.targetName = t.name
	m.targetBaseRevision = t.baseRevision

	m.editImportance = t.importance
	m.editUrgency = t.urgency

	m.dueDateInput = textinput.New()
	m.dueDateInput.Placeholder = "YYYY-MM-DD"
	m.dueDateInput.SetValue(t.dueDate)
	m.priorityFocus = 0
}

func (m *Model) startLinkInput() {
	m.previousView = m.currentView
	m.currentView = ViewLinkInput
	m.linkTextInput = textinput.New()
	m.linkTextInput.Placeholder = "Enter raw content or description to link"
	m.linkTextInput.Focus()
	m.linkTextInput.Width = 60
}

func (m *Model) startMergeSelector(sourceID, sourceName string) {
	m.previousView = m.currentView
	m.currentView = ViewMergeSelector
	m.mergeSourceID = sourceID
	m.mergeSourceName = sourceName

	// filter other dossiers
	m.mergeTargets = nil
	for _, item := range m.items {
		if item.ID != sourceID && item.ID != "" {
			m.mergeTargets = append(m.mergeTargets, item)
		}
	}
	m.mergeCursor = 0
}

func cycleImportance(curr core.Importance, forward bool) core.Importance {
	opts := []core.Importance{core.ImportanceHigh, core.ImportanceLow}
	idx := -1
	for i, o := range opts {
		if o == curr {
			idx = i
			break
		}
	}
	if idx == -1 {
		return core.ImportanceLow
	}
	if forward {
		return opts[(idx+1)%len(opts)]
	}
	return opts[(idx-1+len(opts))%len(opts)]
}

func cycleUrgency(curr core.Urgency, forward bool) core.Urgency {
	opts := []core.Urgency{core.UrgencyHigh, core.UrgencyLow}
	idx := -1
	for i, o := range opts {
		if o == curr {
			idx = i
			break
		}
	}
	if idx == -1 {
		return core.UrgencyLow
	}
	if forward {
		return opts[(idx+1)%len(opts)]
	}
	return opts[(idx-1+len(opts))%len(opts)]
}

func (m *Model) renderMarkdown(content string) string {
	wrapWidth := m.width - 2 // small margin
	if wrapWidth < 40 {
		wrapWidth = 40
	}
	// Rebuild the renderer only when the wrap width changes; constructing one is
	// relatively expensive and renderMarkdown runs on every resize/refresh.
	if m.mdRenderer == nil || m.mdRendererWidth != wrapWidth {
		// Use the default dark style but remove the markdown header prefixes
		cfg := *styles.DefaultStyles["dark"]
		cfg.H1.Prefix = ""
		cfg.H2.Prefix = ""
		cfg.H3.Prefix = ""
		cfg.H4.Prefix = ""
		cfg.H5.Prefix = ""
		cfg.H6.Prefix = ""

		// Reset document colors to inherit terminal defaults (supporting light/dark themes)
		cfg.Document.Color = nil
		cfg.Document.BackgroundColor = nil

		// Signature purple accent for headings
		purpleStr := "99"
		cfg.Heading.Color = &purpleStr
		cfg.Heading.BackgroundColor = nil
		cfg.H1.Color = &purpleStr
		cfg.H1.BackgroundColor = nil
		cfg.H2.Color = &purpleStr
		cfg.H2.BackgroundColor = nil
		cfg.H3.Color = &purpleStr
		cfg.H3.BackgroundColor = nil
		cfg.H4.Color = &purpleStr
		cfg.H4.BackgroundColor = nil
		cfg.H5.Color = &purpleStr
		cfg.H5.BackgroundColor = nil
		cfg.H6.Color = &purpleStr
		cfg.H6.BackgroundColor = nil

		// Make blockquote left border signature purple as well
		cfg.BlockQuote.Color = &purpleStr

		// Cyan for links (standard ANSI 6, theme adaptive)
		cyanStr := "6"
		cfg.Link.Color = &cyanStr
		cfg.LinkText.Color = &cyanStr

		// Inline code: cyan color and no background color to avoid contrast issues on light/dark backgrounds
		cfg.Code.Color = &cyanStr
		cfg.Code.BackgroundColor = nil

		// Gray for horizontal rules (standard ANSI 8)
		grayStr := "8"
		cfg.HorizontalRule.Color = &grayStr

		// Use bw theme for syntax highlighting to avoid hardcoded dark/light colors
		cfg.CodeBlock.Chroma = nil
		cfg.CodeBlock.Theme = "bw"

		r, err := glamour.NewTermRenderer(
			glamour.WithStyles(cfg),
			glamour.WithWordWrap(wrapWidth),
		)
		if err != nil {
			return content
		}
		m.mdRenderer = r
		m.mdRendererWidth = wrapWidth
	}
	if rendered, err := m.mdRenderer.Render(content); err == nil {
		return rendered
	}
	return content
}

// Update handles incoming messages and updates model state.
func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd
	var cmds []tea.Cmd

	switch msg := msg.(type) {
	case tea.KeyMsg:
		// View-specific key overrides
		switch m.currentView {
		case ViewLinkInput:
			switch msg.String() {
			case "esc":
				m.currentView = m.previousView
				return m, nil
			case "enter":
				m.loading = true
				m.err = nil
				return m, m.firstLinkCmd(m.linkTextInput.Value())
			}
			m.linkTextInput, cmd = m.linkTextInput.Update(msg)
			return m, cmd

		case ViewLinkSelector:
			switch msg.String() {
			case "esc":
				m.currentView = ViewDashboard
				return m, nil
			case "up", "k":
				m.linkCursor = (m.linkCursor - 1 + len(m.linkSuggestions)) % len(m.linkSuggestions)
			case "down", "j":
				m.linkCursor = (m.linkCursor + 1) % len(m.linkSuggestions)
			case "enter":
				m.loading = true
				m.err = nil
				return m, m.confirmLinkCmd(m.linkSuggestions[m.linkCursor].ID, m.linkContent)
			}
			return m, nil

		case ViewMergeSelector:
			switch msg.String() {
			case "esc":
				m.currentView = ViewDashboard
				return m, nil
			case "up", "k":
				m.mergeCursor = (m.mergeCursor - 1 + len(m.mergeTargets)) % len(m.mergeTargets)
			case "down", "j":
				m.mergeCursor = (m.mergeCursor + 1) % len(m.mergeTargets)
			case "enter":
				if len(m.mergeTargets) > 0 {
					m.loading = true
					m.err = nil
					m.mergeTargetID = m.mergeTargets[m.mergeCursor].ID
					return m, m.mergeCmd(m.mergeSourceID, m.mergeTargetID, nil)
				}
			}
			return m, nil

		case ViewMergeConflictResolver:
			switch msg.String() {
			case "esc":
				m.currentView = ViewDashboard
				return m, nil
			case "tab", "shift+tab":
				m.conflictResolverCursor = (m.conflictResolverCursor + 1) % 2
			case "enter":
				if m.conflictResolverCursor == 0 {
					m.loading = true
					m.err = nil
					return m, m.mergeCmd(m.mergeSourceID, m.mergeTargetID, []string{m.mergeConflict.ID})
				} else {
					m.currentView = ViewDashboard
					return m, nil
				}
			}
			m.conflictViewport, cmd = m.conflictViewport.Update(msg)
			return m, cmd

		case ViewNextActionEditor:
			switch msg.String() {
			case "esc":
				m.currentView = m.previousView
				return m, nil
			case "enter":
				m.loading = true
				m.err = nil
				return m, m.saveNextActionCmd(m.targetID, m.targetBaseRevision, m.nextActionInput.Value())
			}
			m.nextActionInput, cmd = m.nextActionInput.Update(msg)
			return m, cmd

		case ViewLeadEditor:
			switch msg.String() {
			case "esc":
				m.currentView = m.previousView
				return m, nil
			case "enter":
				m.loading = true
				m.err = nil
				return m, m.saveLeadCmd(m.targetID, m.targetBaseRevision, m.leadInput.Value())
			}
			m.leadInput, cmd = m.leadInput.Update(msg)
			return m, cmd

		case ViewStatusPicker:
			switch msg.String() {
			case "esc":
				m.currentView = m.previousView
				return m, nil
			case "up", "k":
				m.statusCursor = (m.statusCursor - 1 + len(m.statusOptions)) % len(m.statusOptions)
			case "down", "j":
				m.statusCursor = (m.statusCursor + 1) % len(m.statusOptions)
			case "enter":
				m.loading = true
				m.err = nil
				return m, m.setStatusCmd(m.targetID, m.targetBaseRevision, m.statusOptions[m.statusCursor])
			}
			return m, nil

		case ViewPriorityEditor:
			switch msg.String() {
			case "esc":
				m.currentView = m.previousView
				return m, nil
			case "up", "k":
				m.priorityFocus = (m.priorityFocus - 1 + 3) % 3
				if m.priorityFocus == 2 {
					m.dueDateInput.Focus()
				} else {
					m.dueDateInput.Blur()
				}
			case "down", "j", "tab":
				m.priorityFocus = (m.priorityFocus + 1) % 3
				if m.priorityFocus == 2 {
					m.dueDateInput.Focus()
				} else {
					m.dueDateInput.Blur()
				}
			case "shift+tab":
				m.priorityFocus = (m.priorityFocus - 1 + 3) % 3
				if m.priorityFocus == 2 {
					m.dueDateInput.Focus()
				} else {
					m.dueDateInput.Blur()
				}
			case "left", "h":
				if m.priorityFocus == 0 {
					m.editImportance = cycleImportance(m.editImportance, false)
				} else if m.priorityFocus == 1 {
					m.editUrgency = cycleUrgency(m.editUrgency, false)
				}
			case "right", "l":
				if m.priorityFocus == 0 {
					m.editImportance = cycleImportance(m.editImportance, true)
				} else if m.priorityFocus == 1 {
					m.editUrgency = cycleUrgency(m.editUrgency, true)
				}
			case "enter":
				m.loading = true
				m.err = nil
				return m, m.savePriorityCmd(m.targetID, m.targetBaseRevision, m.editImportance, m.editUrgency, m.dueDateInput.Value())
			}

			if m.priorityFocus == 2 {
				m.dueDateInput, cmd = m.dueDateInput.Update(msg)
				return m, cmd
			}
			return m, nil
		}

		// Global keys for Dashboard and Detail Views
		switch msg.String() {
		case "q", "ctrl+c":
			return m, tea.Quit
		case "esc", "backspace":
			if m.currentView == ViewDetail {
				m.currentView = ViewDashboard
				m.warnings = nil
				m.err = nil
				m.table.Focus()
				return m, m.listDossiersCmd()
			}
		case "r":
			m.loading = true
			m.err = nil
			if m.currentView == ViewDetail && m.recallResult.Frontmatter.ID != "" {
				return m, m.recallDossierCmd(m.recallResult.Frontmatter.ID)
			}
			return m, m.listDossiersCmd()
		case "enter":
			if m.currentView == ViewDashboard {
				idx := m.table.Cursor()
				if idx >= 0 && idx < len(m.items) {
					dossierID := m.items[idx].ID
					if dossierID == "" {
						return m, nil // prevent selection of header
					}
					m.loading = true
					m.err = nil
					return m, m.recallDossierCmd(dossierID)
				}
			}
		case "s":
			if t, ok := m.getTargetDossier(); ok {
				m.startEditStatus(t)
				return m, nil
			}
		case "p":
			if t, ok := m.getTargetDossier(); ok && t.id != "" {
				m.startEditPriority(t)
				return m, nil
			}
		case "n":
			if t, ok := m.getTargetDossier(); ok && t.id != "" {
				m.startEditNextAction(t)
				return m, nil
			}
		case "l":
			if t, ok := m.getTargetDossier(); ok && t.id != "" {
				m.startEditLead(t)
				return m, nil
			}
		case "e":
			if m.currentView == ViewDetail && m.recallResult.Frontmatter.ID != "" {
				res, err := m.svc.Path(context.Background(), core.PathReq{ID: m.recallResult.Frontmatter.ID})
				if err != nil {
					m.err = err
					return m, nil
				}
				dossierPath := filepath.Join(res.Data.(string), "dossier.md")
				editor := os.Getenv("EDITOR")
				if editor == "" {
					editor = "nano"
				}
				cmd := exec.Command(editor, dossierPath)
				return m, tea.ExecProcess(cmd, func(err error) tea.Msg {
					return editorFinishedMsg{err: err, id: m.recallResult.Frontmatter.ID}
				})
			}
		case "k":
			if m.currentView == ViewDashboard {
				m.startLinkInput()
				return m, nil
			}
		case "m":
			if m.currentView == ViewDashboard {
				idx := m.table.Cursor()
				if idx >= 0 && idx < len(m.items) {
					m.startMergeSelector(m.items[idx].ID, m.items[idx].Name)
					return m, nil
				}
			}
		}

	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		m.recalculateTableLayout()
		m.recalculateViewportLayout()
		m.recalculateConflictViewportLayout()

		if m.currentView == ViewDetail && m.recallResult.Frontmatter.ID != "" {
			m.viewport.SetContent(m.renderMarkdown(m.recallResult.DistilledState))
		}
		if m.currentView == ViewMergeConflictResolver && m.mergeConflict != nil {
			diffMd := fmt.Sprintf("```diff\n%s\n```", m.mergeConflict.DiffAgainstCurrent)
			m.conflictViewport.SetContent(m.renderMarkdown(diffMd))
		}

	case listDossiersMsg:
		m.loading = false

		sort.Slice(msg, func(i, j int) bool {
			if msg[i].PriorityScore != msg[j].PriorityScore {
				return msg[i].PriorityScore < msg[j].PriorityScore
			}
			d1 := msg[i].DueDate
			d2 := msg[j].DueDate
			if d1 != d2 {
				if d1 == "" {
					return false
				}
				if d2 == "" {
					return true
				}
				return d1 < d2
			}
			return false
		})

		m.items = msg
		m.populateTableRows()
		if len(msg) > 0 {
			m.table.SetCursor(0)
		}

		// Watch every dossier directory so the dashboard live-refreshes on
		// external edits, plus the currently open dossier if it isn't listed.
		var watchPaths []string
		for _, item := range msg {
			watchPaths = append(watchPaths, item.Path)
		}
		if m.currentView == ViewDetail {
			watchPaths = append(watchPaths, m.recallResult.Path)
		}
		m.syncWatches(watchPaths)

	case recallDossierMsg:
		m.loading = false
		if msg.err != nil {
			m.err = msg.err
		} else {
			m.currentView = ViewDetail
			m.recallResult = msg.result
			m.warnings = msg.warnings
			m.viewport.SetContent(m.renderMarkdown(msg.result.DistilledState))
			m.recalculateViewportLayout()
			m.viewport.YOffset = 0

			// Recall returns the dossier's directory path; sync watches including
			// the new path and any currently listed dashboard items to prevent leaks
			// from navigating deep into links.
			var watchPaths []string
			for _, item := range m.items {
				if item.Path != "" {
					watchPaths = append(watchPaths, item.Path)
				}
			}
			watchPaths = append(watchPaths, m.recallResult.Path)
			m.syncWatches(watchPaths)
		}

	case linkResultMsg:
		m.loading = false
		if msg.err != nil {
			// Check if it's a domain error code for ambiguity
			if dErr, ok := msg.err.(*core.DomainError); ok && dErr.Code == core.ErrAmbiguousTarget {
				suggestions, ok := msg.result.Data.([]core.Suggestion)
				if ok && len(suggestions) > 0 {
					m.currentView = ViewLinkSelector
					m.linkSuggestions = suggestions
					m.linkContent = msg.content
					m.linkCursor = 0
					return m, nil
				}
			}
			m.err = msg.err
			m.currentView = ViewDashboard
		} else {
			m.currentView = ViewDashboard
			m.err = nil
			return m, m.listDossiersCmd()
		}

	case linkConfirmResultMsg:
		m.loading = false
		m.currentView = ViewDashboard
		if msg.err != nil {
			m.err = msg.err
		} else {
			m.err = nil
			return m, m.listDossiersCmd()
		}

	case mergeResultMsg:
		m.loading = false
		if msg.err != nil {
			if dErr, ok := msg.err.(*core.DomainError); ok && dErr.Code == core.ErrConflictDetected {
				conflict, ok := msg.result.Data.(*core.Conflict)
				if ok {
					m.currentView = ViewMergeConflictResolver
					m.mergeConflict = conflict
					diffMd := fmt.Sprintf("```diff\n%s\n```", conflict.DiffAgainstCurrent)
					m.conflictViewport.SetContent(m.renderMarkdown(diffMd))
					m.recalculateConflictViewportLayout()
					m.conflictResolverCursor = 0
					return m, nil
				}
			}
			m.err = msg.err
			m.currentView = ViewDashboard
		} else {
			m.currentView = ViewDashboard
			m.err = nil
			// Show success info
			return m, m.listDossiersCmd()
		}

	case mutationResultMsg:
		m.loading = false
		if msg.err != nil {
			m.err = msg.err
			m.currentView = msg.prevView
		} else {
			m.currentView = msg.prevView
			m.err = nil
			if m.currentView == ViewDetail {
				return m, m.recallDossierCmd(msg.targetID)
			} else {
				return m, m.listDossiersCmd()
			}
		}

	case editorFinishedMsg:
		if msg.err != nil {
			m.err = msg.err
			return m, nil
		}
		m.loading = true
		return m, m.recallDossierCmd(msg.id)

	case errMsg:
		m.loading = false
		m.err = msg

	case dossierUpdatedMsg:
		cmds = append(cmds, waitForUpdate(m.updateChan))
		if m.currentView == ViewDetail && m.recallResult.Frontmatter.ID != "" {
			m.loading = true
			cmds = append(cmds, m.recallDossierCmd(m.recallResult.Frontmatter.ID))
		} else if m.currentView == ViewDashboard {
			cmds = append(cmds, m.listDossiersCmd())
		}
	}

	// Update view-specific sub-components
	if m.currentView == ViewDashboard {
		m.table, cmd = m.table.Update(msg)
		cmds = append(cmds, cmd)
	} else if m.currentView == ViewDetail {
		m.viewport, cmd = m.viewport.Update(msg)
		cmds = append(cmds, cmd)
	}

	return m, tea.Batch(cmds...)
}

type editorFinishedMsg struct {
	err error
	id  string
}

// populateTableRows maps items into the table rows.
func (m *Model) tableColumnsConfig() (showPriority, showNextAction, showDue bool) {
	w := m.width
	if w < 44 {
		w = 44
	}
	return w >= 55, w >= 80, w >= 65
}

func (m *Model) populateTableRows() {
	showPriority, showNextAction, showDue := m.tableColumnsConfig()

	rows := make([]table.Row, 0, len(m.items))
	for _, item := range m.items {
		if item.ID == "" {
			row := table.Row{item.Name, "", ""}
			if showPriority {
				row = append(row, "")
			}
			if showNextAction {
				row = append(row, "")
			}
			if showDue {
				row = append(row, "")
			}
			rows = append(rows, row)
			continue
		}

		leadStr := item.Lead
		if leadStr != "" {
			parts := strings.Fields(leadStr)
			if len(parts) > 1 {
				leadStr = parts[0] + " " + string(parts[len(parts)-1][0])
			}
		}

		statusStr := item.Status
		var priorityStr string
		switch item.PriorityScore {
		case 1:
			priorityStr = "1. Do"
		case 2:
			priorityStr = "2. Plan"
		case 3:
			priorityStr = "3. Delegate"
		case 4:
			priorityStr = "4. Delete"
		default:
			priorityStr = strconv.Itoa(item.PriorityScore)
		}

		dueStr := ""
		if item.DueDate != "" {
			t, err := time.Parse("2006-01-02", item.DueDate)
			if err == nil {
				dueStr = t.Format("01/02")
			} else {
				dueStr = item.DueDate
			}
		}

		row := table.Row{
			item.Name,
			statusStr,
			leadStr,
		}
		if showPriority {
			row = append(row, priorityStr)
		}
		if showNextAction {
			row = append(row, item.NextAction)
		}
		if showDue {
			row = append(row, dueStr)
		}
		rows = append(rows, row)
	}
	m.table.SetRows(rows)
}

// recalculateTableLayout fits the table to the screen size.
func (m *Model) recalculateTableLayout() {
	tableHeight := m.height - 7
	if tableHeight < 3 {
		tableHeight = 3
	}
	m.table.SetHeight(tableHeight)

	showPriority, showNextAction, showDue := m.tableColumnsConfig()

	cols := []table.Column{
		{Title: "Name", Width: 18},
		{Title: "Status", Width: 8},
		{Title: "Lead", Width: 8},
	}
	usedWidth := 18 + 8 + 8
	numCols := 3

	if showPriority {
		usedWidth += 12
		numCols++
	}
	if showDue {
		usedWidth += 8
		numCols++
	}
	if showNextAction {
		numCols++
	}

	overhead := (numCols * 2) + (numCols - 1)

	if showPriority {
		cols = append(cols, table.Column{Title: "Priority", Width: 12})
	}
	if showNextAction {
		nextActionWidth := m.width - usedWidth - overhead
		if nextActionWidth < 12 {
			nextActionWidth = 12
		}
		cols = append(cols, table.Column{Title: "Next Action", Width: nextActionWidth})
	}
	if showDue {
		cols = append(cols, table.Column{Title: "Due", Width: 8})
	}

	m.table.SetRows(nil) // Prevent panic from bubbles/table looping old rows against new columns
	m.table.SetColumns(cols)
	m.populateTableRows()
}

// recalculateViewportLayout fits the viewport to the screen.
func (m *Model) recalculateViewportLayout() {
	m.viewport.Width = m.width
	m.viewport.Height = m.height - 13
	if m.viewport.Height < 3 {
		m.viewport.Height = 3
	}
}

// recalculateConflictViewportLayout fits the conflict viewport to the screen.
func (m *Model) recalculateConflictViewportLayout() {
	m.conflictViewport.Width = m.width - 6
	m.conflictViewport.Height = m.height - 17
	if m.conflictViewport.Height < 3 {
		m.conflictViewport.Height = 3
	}
}

func (m Model) renderStatusPicker() string {
	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("Select new status for %s:\n\n", m.targetName))

	for i, opt := range m.statusOptions {
		cursor := "  "
		if i == m.statusCursor {
			cursor = "> "
		}

		statusStr := string(opt)
		var style lipgloss.Style
		switch opt {
		case core.StatusActive:
			style = statusActiveStyle
		case core.StatusWaiting:
			style = statusWaitingStyle
		case core.StatusBlocked:
			style = statusBlockedStyle
		case core.StatusResolved:
			style = statusResolvedStyle
		case core.StatusArchived:
			style = statusArchivedStyle
		}

		if i == m.statusCursor {
			sb.WriteString(focusedItemStyle.Render(fmt.Sprintf("%s%s", cursor, statusStr)))
		} else {
			sb.WriteString(fmt.Sprintf("%s%s", cursor, style.Render(statusStr)))
		}
		sb.WriteString("\n")
	}

	return editorBoxStyle.Render(sb.String())
}

func (m Model) renderNextActionEditor() string {
	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("Edit Next Action for %s:\n\n", m.targetName))
	sb.WriteString(m.nextActionInput.View())
	sb.WriteString("\n\n")
	sb.WriteString("press enter to save • esc to cancel")
	return editorBoxStyle.Render(sb.String())
}

func (m Model) renderLeadEditor() string {
	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("Assigning %s\n\n", lipgloss.NewStyle().Foreground(vibrantGreen).Bold(true).Render(m.targetName)))
	sb.WriteString("Lead (full name):\n")
	sb.WriteString(m.leadInput.View())
	sb.WriteString("\n\n")
	sb.WriteString("press enter to save • esc to cancel")
	return editorBoxStyle.Render(sb.String())
}

func (m Model) renderPriorityEditor() string {
	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("Edit Priority & Due Date for %s:\n\n", m.targetName))

	// Importance
	sb.WriteString(" Importance: ")
	impOpts := []core.Importance{core.ImportanceHigh, core.ImportanceLow}
	var impStr []string
	for _, o := range impOpts {
		val := string(o)
		if o == m.editImportance {
			val = fmt.Sprintf("[%s]", val)
			val = activeOptionStyle.Render(val)
		} else {
			val = fmt.Sprintf(" %s ", val)
		}
		impStr = append(impStr, val)
	}
	importanceRow := strings.Join(impStr, " ")
	if m.priorityFocus == 0 {
		sb.WriteString(focusedItemStyle.Render(importanceRow))
	} else {
		sb.WriteString(importanceRow)
	}
	sb.WriteString("\n\n")

	// Urgency
	sb.WriteString(" Urgency:    ")
	urgOpts := []core.Urgency{core.UrgencyHigh, core.UrgencyLow}
	var urgStr []string
	for _, o := range urgOpts {
		val := string(o)
		if o == m.editUrgency {
			val = fmt.Sprintf("[%s]", val)
			val = activeOptionStyle.Render(val)
		} else {
			val = fmt.Sprintf(" %s ", val)
		}
		urgStr = append(urgStr, val)
	}
	urgencyRow := strings.Join(urgStr, " ")
	if m.priorityFocus == 1 {
		sb.WriteString(focusedItemStyle.Render(urgencyRow))
	} else {
		sb.WriteString(urgencyRow)
	}
	sb.WriteString("\n\n")

	// Due Date
	sb.WriteString(" Due Date:   ")
	if m.priorityFocus == 2 {
		sb.WriteString(m.dueDateInput.View())
	} else {
		val := m.dueDateInput.Value()
		if val == "" {
			val = "(empty)"
		}
		sb.WriteString(metaValueStyle.Render(val))
	}
	sb.WriteString("\n\n")

	// Buttons
	sb.WriteString("press enter to save • esc to cancel")

	return editorBoxStyle.Render(sb.String())
}

func (m Model) renderLinkInput() string {
	var sb strings.Builder
	sb.WriteString("Link Session Content:\n\n")
	sb.WriteString("Enter raw content or description to link to a dossier:\n\n")
	sb.WriteString(m.linkTextInput.View())
	sb.WriteString("\n\n")
	sb.WriteString("press enter to analyze matches • esc to cancel")
	return editorBoxStyle.Render(sb.String())
}

func (m Model) renderLinkSelector() string {
	var sb strings.Builder
	sb.WriteString("Ambiguous Link Targets:\n")
	sb.WriteString("Multiple dossiers match. Select target to confirm link:\n\n")

	for i, sug := range m.linkSuggestions {
		cursor := "  "
		if i == m.linkCursor {
			cursor = "> "
		}

		sugLine := fmt.Sprintf("%-20s (Confidence: %-7s) - Reason: %s", sug.Name, sug.Confidence, sug.Reason)
		if i == m.linkCursor {
			sb.WriteString(focusedItemStyle.Render(cursor + sugLine))
		} else {
			sb.WriteString(cursor + sugLine)
		}
		sb.WriteString("\n")
	}

	sb.WriteString("\n")
	sb.WriteString("press enter to confirm • esc to cancel")
	return editorBoxStyle.Render(sb.String())
}

func (m Model) renderMergeSelector() string {
	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("Merge Dossier: %s (Source)\n", m.mergeSourceName))
	sb.WriteString("Choose the surviving TARGET dossier to merge into:\n\n")

	if len(m.mergeTargets) == 0 {
		sb.WriteString(" No other dossiers available to merge into.\n")
	} else {
		for i, tgt := range m.mergeTargets {
			cursor := "  "
			if i == m.mergeCursor {
				cursor = "> "
			}

			tgtLine := fmt.Sprintf("%s (%s) - status: %s", tgt.Name, tgt.ID, tgt.Status)
			if i == m.mergeCursor {
				sb.WriteString(focusedItemStyle.Render(cursor + tgtLine))
			} else {
				sb.WriteString(cursor + tgtLine)
			}
			sb.WriteString("\n")
		}
	}

	sb.WriteString("\n")
	sb.WriteString("press enter to perform merge • esc to cancel")
	return editorBoxStyle.Render(sb.String())
}

func (m Model) renderMergeConflictResolver() string {
	var sb strings.Builder
	sb.WriteString(warningStyle.Render("⚡ MERGE CONFLICT DETECTED\n"))
	sb.WriteString("Divergent distilled states or statuses cannot be merged automatically.\n")
	sb.WriteString("Review the diff below representing incoming source changes against target:\n\n")

	sb.WriteString(m.conflictViewport.View())
	sb.WriteString("\n\n")

	sb.WriteString(subtitleStyle.Render("ℹ Note: Source dossier files are retained and archived, never deleted.\n\n"))

	resolveBtn := "[ Resolve Conflict & Force Merge ]"
	if m.conflictResolverCursor == 0 {
		resolveBtn = focusedItemStyle.Render(resolveBtn)
	}

	cancelBtn := "[ Cancel Merge ]"
	if m.conflictResolverCursor == 1 {
		cancelBtn = focusedItemStyle.Render(cancelBtn)
	}

	sb.WriteString(fmt.Sprintf(" %s   %s", resolveBtn, cancelBtn))

	return editorBoxStyle.Render(sb.String())
}

// View renders the screen based on state.
func (m Model) View() string {
	if m.width == 0 || m.height == 0 {
		return "Initializing TUI..."
	}

	var sb strings.Builder

	// 1. Header Banner
	sb.WriteString(titleStyle.Render(" DOSSIER TUI "))
	sb.WriteString("\n")

	// Check if there is a primary error message to show
	if m.err != nil {
		sb.WriteString(errorStyle.Render(fmt.Sprintf(" Error: %v\n\n", m.err)))
	}

	switch m.currentView {
	case ViewDashboard:
		sb.WriteString(subtitleStyle.Render(" Durable memory layer for agentic workflows — Dashboard"))
		sb.WriteString("\n\n")

		if m.loading && len(m.items) == 0 {
			sb.WriteString(" Loading dossiers...\n")
		} else {
			sb.WriteString(m.table.View())
			sb.WriteString("\n")
		}

	case ViewDetail:
		sb.WriteString(subtitleStyle.Render(" Durable memory layer for agentic workflows — Recall Detail"))
		sb.WriteString("\n\n")

		fm := m.recallResult.Frontmatter
		score := core.CalculatePriorityScore(fm, time.Now())

		targetTokens := fm.TokenTarget
		if targetTokens == 0 {
			targetTokens = 100000
		}

		lblStyle := metaLabelStyle.Copy().
			Width(10).
			Align(lipgloss.Right).
			MarginRight(1)

		valWidth := m.width - 12
		if valWidth < 10 {
			valWidth = 10
		}
		valStyle := metaValueStyle.Copy().Width(valWidth)

		renderRow := func(label, value string) string {
			return lipgloss.JoinHorizontal(lipgloss.Top,
				lblStyle.Render(label),
				valStyle.Render(value),
			) + "\n"
		}

		col1ValWidth := 20
		col1ValStyle := metaValueStyle.Copy().Width(col1ValWidth)

		col2ValWidth := m.width - 12 - 11 - col1ValWidth
		if col2ValWidth < 10 {
			col2ValWidth = 10
		}
		col2ValStyle := metaValueStyle.Copy().Width(col2ValWidth)

		renderTwoCols := func(l1, v1, l2, v2 string) string {
			if m.width < 90 {
				return renderRow(l1, v1) + renderRow(l2, v2)
			}
			col1 := lipgloss.JoinHorizontal(lipgloss.Top,
				lblStyle.Render(l1),
				col1ValStyle.Render(v1),
			)
			col2 := lipgloss.JoinHorizontal(lipgloss.Top,
				lblStyle.Render(l2),
				col2ValStyle.Render(v2),
			)
			return lipgloss.JoinHorizontal(lipgloss.Top, col1, col2) + "\n"
		}

		// Metadata Block
		leadLabel := fm.Lead
		if leadLabel == "" {
			leadLabel = "Unassigned (Me)"
		}

		sb.WriteString(renderRow("Dossier:", fm.Name))
		sb.WriteString(renderTwoCols(
			"Status:", string(fm.Status),
			"Lead:", leadLabel,
		))
		sb.WriteString(renderRow("Priority:", fmt.Sprintf("Score %d (Importance: %s, Urgency: %s)", score, fm.Importance, fm.Urgency)))
		sb.WriteString(renderRow("Tokens:", fmt.Sprintf("%d / %d", m.recallResult.TokenEstimate, targetTokens)))
		sb.WriteString(renderRow("Next:", fm.NextAction))

		sb.WriteString(lipgloss.NewStyle().Foreground(darkGray).Render(strings.Repeat("─", m.width)))
		sb.WriteString("\n")

		// Scrollable viewport
		sb.WriteString(m.viewport.View())
		sb.WriteString("\n")

	case ViewStatusPicker:
		sb.WriteString(subtitleStyle.Render(" Durable memory layer for agentic workflows — Update Status"))
		sb.WriteString("\n\n")
		sb.WriteString(m.renderStatusPicker())
		sb.WriteString("\n")

	case ViewNextActionEditor:
		sb.WriteString(subtitleStyle.Render(" Durable memory layer for agentic workflows — Update Next Action"))
		sb.WriteString("\n\n")
		sb.WriteString(m.renderNextActionEditor())
		sb.WriteString("\n")

	case ViewLeadEditor:
		sb.WriteString(subtitleStyle.Render(" Durable memory layer for agentic workflows — Update Lead"))
		sb.WriteString("\n\n")
		sb.WriteString(m.renderLeadEditor())
		sb.WriteString("\n")

	case ViewPriorityEditor:
		sb.WriteString(subtitleStyle.Render(" Durable memory layer for agentic workflows — Update Priority"))
		sb.WriteString("\n\n")
		sb.WriteString(m.renderPriorityEditor())
		sb.WriteString("\n")

	case ViewLinkInput:
		sb.WriteString(subtitleStyle.Render(" Durable memory layer for agentic workflows — Link Content"))
		sb.WriteString("\n\n")
		sb.WriteString(m.renderLinkInput())
		sb.WriteString("\n")

	case ViewLinkSelector:
		sb.WriteString(subtitleStyle.Render(" Durable memory layer for agentic workflows — Resolve Ambiguous Link"))
		sb.WriteString("\n\n")
		sb.WriteString(m.renderLinkSelector())
		sb.WriteString("\n")

	case ViewMergeSelector:
		sb.WriteString(subtitleStyle.Render(" Durable memory layer for agentic workflows — Merge Dossier"))
		sb.WriteString("\n\n")
		sb.WriteString(m.renderMergeSelector())
		sb.WriteString("\n")

	case ViewMergeConflictResolver:
		sb.WriteString(subtitleStyle.Render(" Durable memory layer for agentic workflows — Resolve Merge Conflict"))
		sb.WriteString("\n\n")
		sb.WriteString(m.renderMergeConflictResolver())
		sb.WriteString("\n")
	}

	// 3. Footer / Help area
	sb.WriteString("\n")
	var footerParts []string
	if len(m.warnings) > 0 {
		for _, w := range m.warnings {
			footerParts = append(footerParts, warningStyle.Render(fmt.Sprintf("⚠ %s", w)))
		}
	}

	keyHelp := "↑/↓: select • enter: detail • s: status • l: lead • p: priority • n: next action • k: link • m: merge • q: quit"
	switch m.currentView {
	case ViewDetail:
		keyHelp = "↑/↓/pgup/pgdn: scroll • s: status • l: lead • p: priority • n: next action • esc: back • q: quit"
	case ViewStatusPicker:
		keyHelp = "↑/↓: select status • enter: confirm • esc: cancel"
	case ViewNextActionEditor:
		keyHelp = "enter: save next action • esc: cancel"
	case ViewLeadEditor:
		keyHelp = "enter: save lead • esc: cancel"
	case ViewPriorityEditor:
		keyHelp = "↑/↓: focus • ←/→: cycle priority • enter: cycle/save • esc: cancel"
	case ViewLinkInput:
		keyHelp = "enter: analyze matching candidates • esc: cancel"
	case ViewLinkSelector:
		keyHelp = "↑/↓: select target dossier • enter: link • esc: cancel"
	case ViewMergeSelector:
		keyHelp = "↑/↓: select target dossier • enter: merge • esc: cancel"
	case ViewMergeConflictResolver:
		keyHelp = "↑/↓/pgup/pgdn: scroll diff • tab: switch button • enter: confirm • esc: cancel"
	}
	footerParts = append(footerParts, keyHelp)

	sb.WriteString(footerStyle.Width(m.width).Render(strings.Join(footerParts, " │ ")))

	return sb.String()
}

// Run sets up the program, enters the alt-screen, and executes.
//
// NOTE (ADR 0004): the TUI does not resolve or carry a session identity. It is a
// read/edit viewer over the dossier store; the per-session "active" binding (Switch)
// is intentionally not exposed here — see ADR 0004 and BUILD-DECISIONS B9.
func Run(ctx context.Context, svc *core.Service) error {
	m := NewModel(svc)
	if m.watcher != nil {
		defer m.watcher.Close()
	}
	p := tea.NewProgram(
		m,
		tea.WithAltScreen(),
		tea.WithContext(ctx),
	)
	_, err := p.Run()
	return err
}
