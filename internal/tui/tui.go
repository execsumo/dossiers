package tui

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"time"

	"dossier/internal/core"

	"github.com/charmbracelet/bubbles/table"
	"github.com/charmbracelet/bubbles/textinput"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// View represents the current active TUI screen view.
type View int

const (
	ViewDashboard View = iota
	ViewDetail
	ViewStatusPicker
	ViewNextActionEditor
	ViewPriorityEditor
	ViewLinkInput
	ViewLinkSelector
	ViewMergeSelector
	ViewMergeConflictResolver
)

// Styling tokens
var (
	purple       = lipgloss.Color("99")
	lightGray    = lipgloss.Color("250")
	darkGray     = lipgloss.Color("237")
	vibrantGreen = lipgloss.Color("42")
	vibrantRed   = lipgloss.Color("196")
	vibrantBlue  = lipgloss.Color("33")
	warningGold  = lipgloss.Color("208")

	titleStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("229")).
			Background(purple).
			Padding(0, 2).
			Bold(true)

	subtitleStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("244")).
			Italic(true)

	activeDossierStyle = lipgloss.NewStyle().
				Foreground(vibrantGreen).
				Bold(true)

	sessionStyle = lipgloss.NewStyle().
			Foreground(vibrantBlue).
			Bold(true)

	headerStyle = lipgloss.NewStyle().
			Foreground(purple).
			Bold(true)

	footerStyle = lipgloss.NewStyle().
			Background(darkGray).
			Foreground(lightGray).
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

	metaValueStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("252"))

	statusActiveStyle   = lipgloss.NewStyle().Foreground(vibrantGreen).Bold(true)
	statusWaitingStyle  = lipgloss.NewStyle().Foreground(warningGold)
	statusBlockedStyle  = lipgloss.NewStyle().Foreground(vibrantRed).Bold(true)
	statusResolvedStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("244"))
	statusArchivedStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("240"))

	editorBoxStyle = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(purple).
			Padding(1, 2).
			Margin(1, 0)

	focusedItemStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color("229")).
				Background(purple).
				Bold(true).
				Padding(0, 1)

	activeOptionStyle = lipgloss.NewStyle().
				Foreground(vibrantGreen).
				Bold(true)
)

// Messages
type listDossiersMsg []core.ListItem
type activeDossierMsg *core.SessionBinding
type recallDossierMsg struct {
	id       string
	result   core.RecallResult
	err      error
	warnings []core.Warning
}
type switchActiveMsg struct {
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

type targetDossier struct {
	id           string
	name         string
	status       core.Status
	importance   core.Importance
	urgency      core.Urgency
	dueDate      string
	nextAction   string
	baseRevision core.Revision
}

// Model holds the application state.
type Model struct {
	svc         *core.Service
	sessionID   string
	currentView View

	// Data
	items        []core.ListItem
	activeID     string
	activeName   string
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

	// Priority Editor view state
	priorityFocus  int // 0 = Importance, 1 = Urgency, 2 = Due Date, 3 = Save, 4 = Cancel
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
}

// NewModel instantiates the root TUI model.
func NewModel(svc *core.Service, sessionID string) Model {
	// Initialize default empty table
	columns := []table.Column{
		{Title: "A", Width: 2},
		{Title: "Name", Width: 22},
		{Title: "Status", Width: 10},
		{Title: "Priority", Width: 8},
		{Title: "Next Action", Width: 35},
		{Title: "Staleness", Width: 10},
	}

	t := table.New(
		table.WithColumns(columns),
		table.WithFocused(true),
	)

	s := table.DefaultStyles()
	s.Header = s.Header.
		BorderStyle(lipgloss.NormalBorder()).
		BorderForeground(lipgloss.Color("240")).
		BorderBottom(true).
		Bold(true)
	s.Selected = s.Selected.
		Foreground(lipgloss.Color("229")).
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

	return Model{
		svc:              svc,
		sessionID:        sessionID,
		currentView:      ViewDashboard,
		table:            t,
		viewport:         vp,
		conflictViewport: cvp,
		loading:          true,
		statusOptions:    statusOptions,
	}
}

// Init initializes the tea program, triggering initial loads.
func (m Model) Init() tea.Cmd {
	return tea.Batch(
		m.listDossiersCmd(),
		m.getActiveDossierCmd(),
	)
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

// getActiveDossierCmd fetches the currently active session binding.
func (m Model) getActiveDossierCmd() tea.Cmd {
	return func() tea.Msg {
		res, err := m.svc.Active(context.Background(), core.ActiveReq{SessionID: m.sessionID})
		if err != nil {
			return activeDossierMsg(nil)
		}
		binding, ok := res.Data.(*core.SessionBinding)
		if !ok {
			return activeDossierMsg(nil)
		}
		return activeDossierMsg(binding)
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

func (m Model) switchActiveCmd(id string) tea.Cmd {
	return func() tea.Msg {
		res, err := m.svc.Switch(context.Background(), core.SwitchReq{
			ID:        id,
			SessionID: m.sessionID,
		})
		if err != nil {
			return switchActiveMsg{id: id, err: err}
		}
		recallRes, ok := res.Data.(core.RecallResult)
		if !ok {
			return switchActiveMsg{id: id, err: fmt.Errorf("invalid switch data type")}
		}
		return switchActiveMsg{
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

func (m Model) setStatusCmd(id string, status core.Status) tea.Cmd {
	return func() tea.Msg {
		_, err := m.svc.SetStatus(context.Background(), core.SetStatusReq{
			ID:     id,
			Status: status,
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
		if item.ID != sourceID {
			m.mergeTargets = append(m.mergeTargets, item)
		}
	}
	m.mergeCursor = 0
}

func cycleImportance(curr core.Importance, forward bool) core.Importance {
	opts := []core.Importance{core.ImportanceHigh, core.ImportanceMedium, core.ImportanceLow}
	idx := -1
	for i, o := range opts {
		if o == curr {
			idx = i
			break
		}
	}
	if idx == -1 {
		return core.ImportanceMedium
	}
	if forward {
		return opts[(idx+1)%len(opts)]
	}
	return opts[(idx-1+len(opts))%len(opts)]
}

func cycleUrgency(curr core.Urgency, forward bool) core.Urgency {
	opts := []core.Urgency{core.UrgencyHigh, core.UrgencyMedium, core.UrgencyLow}
	idx := -1
	for i, o := range opts {
		if o == curr {
			idx = i
			break
		}
	}
	if idx == -1 {
		return core.UrgencyMedium
	}
	if forward {
		return opts[(idx+1)%len(opts)]
	}
	return opts[(idx-1+len(opts))%len(opts)]
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
				return m, m.setStatusCmd(m.targetID, m.statusOptions[m.statusCursor])
			}
			return m, nil

		case ViewPriorityEditor:
			switch msg.String() {
			case "esc":
				m.currentView = m.previousView
				return m, nil
			case "up", "k":
				m.priorityFocus = (m.priorityFocus - 1 + 5) % 5
				if m.priorityFocus == 2 {
					m.dueDateInput.Focus()
				} else {
					m.dueDateInput.Blur()
				}
			case "down", "j", "tab":
				m.priorityFocus = (m.priorityFocus + 1) % 5
				if m.priorityFocus == 2 {
					m.dueDateInput.Focus()
				} else {
					m.dueDateInput.Blur()
				}
			case "shift+tab":
				m.priorityFocus = (m.priorityFocus - 1 + 5) % 5
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
				if m.priorityFocus == 0 {
					m.editImportance = cycleImportance(m.editImportance, true)
				} else if m.priorityFocus == 1 {
					m.editUrgency = cycleUrgency(m.editUrgency, true)
				} else if m.priorityFocus == 2 {
					m.priorityFocus = 3
					m.dueDateInput.Blur()
				} else if m.priorityFocus == 3 {
					m.loading = true
					m.err = nil
					return m, m.savePriorityCmd(m.targetID, m.targetBaseRevision, m.editImportance, m.editUrgency, m.dueDateInput.Value())
				} else if m.priorityFocus == 4 {
					m.currentView = m.previousView
					return m, nil
				}
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
			return m, tea.Batch(m.listDossiersCmd(), m.getActiveDossierCmd())
		case "enter":
			if m.currentView == ViewDashboard {
				idx := m.table.Cursor()
				if idx >= 0 && idx < len(m.items) {
					dossierID := m.items[idx].ID
					m.loading = true
					m.err = nil
					return m, m.recallDossierCmd(dossierID)
				}
			}
		case "a":
			if t, ok := m.getTargetDossier(); ok {
				m.loading = true
				m.err = nil
				return m, m.switchActiveCmd(t.id)
			}
		case "s":
			if t, ok := m.getTargetDossier(); ok {
				m.startEditStatus(t)
				return m, nil
			}
		case "p":
			if t, ok := m.getTargetDossier(); ok {
				m.startEditPriority(t)
				return m, nil
			}
		case "n":
			if t, ok := m.getTargetDossier(); ok {
				m.startEditNextAction(t)
				return m, nil
			}
		case "l":
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

	case listDossiersMsg:
		m.loading = false
		m.items = msg
		m.populateTableRows()

	case activeDossierMsg:
		if msg != nil {
			m.activeID = msg.DossierID
			m.updateActiveName()
		} else {
			m.activeID = ""
			m.activeName = "None"
		}
		m.populateTableRows()

	case recallDossierMsg:
		m.loading = false
		if msg.err != nil {
			m.err = msg.err
		} else {
			m.currentView = ViewDetail
			m.recallResult = msg.result
			m.warnings = msg.warnings
			m.viewport.SetContent(msg.result.DistilledState)
			m.recalculateViewportLayout()
			m.viewport.YOffset = 0
		}

	case switchActiveMsg:
		m.loading = false
		if msg.err != nil {
			m.err = msg.err
		} else {
			m.activeID = msg.id
			m.activeName = msg.result.Frontmatter.Name
			m.warnings = msg.warnings
			m.populateTableRows()
			return m, tea.Batch(m.listDossiersCmd(), m.getActiveDossierCmd())
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
			return m, tea.Batch(m.listDossiersCmd(), m.getActiveDossierCmd())
		}

	case linkConfirmResultMsg:
		m.loading = false
		m.currentView = ViewDashboard
		if msg.err != nil {
			m.err = msg.err
		} else {
			m.err = nil
			return m, tea.Batch(m.listDossiersCmd(), m.getActiveDossierCmd())
		}

	case mergeResultMsg:
		m.loading = false
		if msg.err != nil {
			if dErr, ok := msg.err.(*core.DomainError); ok && dErr.Code == core.ErrConflictDetected {
				conflict, ok := msg.result.Data.(*core.Conflict)
				if ok {
					m.currentView = ViewMergeConflictResolver
					m.mergeConflict = conflict
					m.conflictViewport.SetContent(conflict.DiffAgainstCurrent)
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
			return m, tea.Batch(m.listDossiersCmd(), m.getActiveDossierCmd())
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
				return m, tea.Batch(m.listDossiersCmd(), m.getActiveDossierCmd())
			}
		}

	case errMsg:
		m.loading = false
		m.err = msg
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

// updateActiveName resolves the active name from items if available.
func (m *Model) updateActiveName() {
	if m.activeID == "" {
		m.activeName = "None"
		return
	}
	for _, item := range m.items {
		if item.ID == m.activeID {
			m.activeName = item.Name
			return
		}
	}
	m.activeName = m.activeID
}

// populateTableRows maps items into the table rows.
func (m *Model) populateTableRows() {
	m.updateActiveName()
	rows := make([]table.Row, len(m.items))
	for i, item := range m.items {
		activeMarker := "  "
		if item.ID == m.activeID {
			activeMarker = "★ "
		}

		statusStr := item.Status
		priorityStr := strconv.Itoa(item.PriorityScore)
		stalenessStr := fmt.Sprintf("%dd ago", item.StalenessDays)
		if item.StalenessDays == 0 {
			stalenessStr = "today"
		}

		rows[i] = table.Row{
			activeMarker,
			item.Name,
			statusStr,
			priorityStr,
			item.NextAction,
			stalenessStr,
		}
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

	w := m.width
	if w < 87 {
		w = 87
	}

	nextActionWidth := w - (2 + 22 + 10 + 8 + 10 + 10)
	if nextActionWidth < 15 {
		nextActionWidth = 15
	}

	m.table.SetColumns([]table.Column{
		{Title: "A", Width: 2},
		{Title: "Name", Width: 22},
		{Title: "Status", Width: 10},
		{Title: "Priority", Width: 8},
		{Title: "Next Action", Width: nextActionWidth},
		{Title: "Staleness", Width: 10},
	})
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

func (m Model) renderPriorityEditor() string {
	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("Edit Priority & Due Date for %s:\n\n", m.targetName))

	// Importance
	sb.WriteString(" Importance: ")
	impOpts := []core.Importance{core.ImportanceHigh, core.ImportanceMedium, core.ImportanceLow}
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
	urgOpts := []core.Urgency{core.UrgencyHigh, core.UrgencyMedium, core.UrgencyLow}
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
	saveBtn := "[ Save ]"
	if m.priorityFocus == 3 {
		saveBtn = focusedItemStyle.Render(saveBtn)
	}

	cancelBtn := "[ Cancel ]"
	if m.priorityFocus == 4 {
		cancelBtn = focusedItemStyle.Render(cancelBtn)
	}

	sb.WriteString(fmt.Sprintf(" %s   %s", saveBtn, cancelBtn))

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
	headerText := fmt.Sprintf(" DOSSIER TUI │ Session: %s │ Active: %s ", m.sessionID, m.activeName)
	sb.WriteString(titleStyle.Render(headerText))
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

		// Metadata Block
		sb.WriteString(metaLabelStyle.Render(" Dossier: "))
		sb.WriteString(metaValueStyle.Render(fmt.Sprintf("%s (%s)\n", fm.Name, fm.ID)))

		sb.WriteString(metaLabelStyle.Render(" Status:  "))
		sb.WriteString(metaValueStyle.Render(fmt.Sprintf("%-15s", fm.Status)))
		sb.WriteString(metaLabelStyle.Render(" Priority: "))
		sb.WriteString(metaValueStyle.Render(fmt.Sprintf("Score %d (Importance: %s, Urgency: %s)\n", score, fm.Importance, fm.Urgency)))

		targetTokens := fm.TokenTarget
		if targetTokens == 0 {
			targetTokens = 100000
		}
		sb.WriteString(metaLabelStyle.Render(" Tokens:  "))
		sb.WriteString(metaValueStyle.Render(fmt.Sprintf("%d / %d", m.recallResult.TokenEstimate, targetTokens)))
		sb.WriteString(metaLabelStyle.Render("       Revision: "))
		sb.WriteString(metaValueStyle.Render(fmt.Sprintf("%s\n", m.recallResult.Revision)))

		sb.WriteString(metaLabelStyle.Render(" Next:    "))
		sb.WriteString(metaValueStyle.Render(fmt.Sprintf("%s\n", fm.NextAction)))

		sb.WriteString(lipgloss.NewStyle().Foreground(lipgloss.Color("240")).Render(strings.Repeat("─", m.width)))
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

	keyHelp := "↑/↓: select • enter: detail • s: status • p: priority • n: next action • a: active • l: link • m: merge • q: quit"
	switch m.currentView {
	case ViewDetail:
		keyHelp = "↑/↓/pgup/pgdn: scroll • s: status • p: priority • n: next action • a: active • esc: back • q: quit"
	case ViewStatusPicker:
		keyHelp = "↑/↓: select status • enter: confirm • esc: cancel"
	case ViewNextActionEditor:
		keyHelp = "enter: save next action • esc: cancel"
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
func Run(ctx context.Context, svc *core.Service, sessionID string) error {
	p := tea.NewProgram(
		NewModel(svc, sessionID),
		tea.WithAltScreen(),
		tea.WithContext(ctx),
	)
	_, err := p.Run()
	return err
}
