package ui

import (
	"bytes"
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/Gostatsog/dockerNav/internal/client"
	"github.com/Gostatsog/dockerNav/pkg/formatter"
	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/list"
	"github.com/charmbracelet/bubbles/spinner"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/network"
)

type Port struct {
	// Host IP address that the container's port is mapped to
	IP string `json:"IP,omitempty"`

	// Port on the container
	// Required: true
	PrivatePort uint16 `json:"PrivatePort"`

	// Port exposed on the host
	PublicPort uint16 `json:"PublicPort,omitempty"`

	// type
	// Required: true
	Type string `json:"Type"`
}

type MountPoint struct {
	Type        string `json:"Type"`
	Name        string `json:"Name"`
	Source      string `json:"Source"`
	Destination string `json:"Destination"`
	Driver      string `json:"Driver"`
	Mode        string `json:"Mode"`
	RW          bool   `json:"RW"`
	Propagation string `json:"Propagation"`
}

type NetworkSettingsSummary struct {
	Networks map[string]*network.EndpointSettings
}

type Summary struct {
	ID         string            `json:"Id"`
	Names      []string          `json:"Names"`
	Image      string            `json:"Image"`
	ImageID    string            `json:"ImageID"`
	Command    string            `json:"Command"`
	Created    int64             `json:"Created"`
	Ports      []Port            `json:"Ports"`
	SizeRw     int64             `json:"SizeRw,omitempty"`
	SizeRootFs int64             `json:"SizeRootFs,omitempty"`
	Labels     map[string]string `json:"Labels"`
	State      string            `json:"State"`
	Status     string            `json:"Status"`
	HostConfig struct {
		NetworkMode string            `json:"NetworkMode,omitempty"`
		Annotations map[string]string `json:"Annotations,omitempty"`
	} `json:"HostConfig"`
	NetworkSettings *NetworkSettingsSummary `json:"NetworkSettings"`
	Mounts          []MountPoint            `json:"Mounts"`
}

// ContainerListMsg carries container data after fetching
type ContainerListMsg struct {
	Containers []Summary
	Error      error
}

// ContainerActionMsg carries results of container actions
type ContainerActionMsg struct {
	Action      string
	ContainerID string
	Error       error
}

// ContainerLogsMsg carries container logs data
type ContainerLogsMsg struct {
	Logs  string
	Error error
}

// ContainerItem represents a container in the list
type ContainerItem struct {
	container Summary
	title     string
	desc      string
}

// FilterValue implements list.Item interface
func (i ContainerItem) FilterValue() string { return i.title }

// Title returns the title for the list item
func (i ContainerItem) Title() string { return i.title }

// Description returns the description for the list item
func (i ContainerItem) Description() string { return i.desc }

// ContainerKeyMap defines keybindings for container operations
type ContainerKeyMap struct {
	Refresh  key.Binding
	Logs     key.Binding
	Stop     key.Binding
	Start    key.Binding
	Restart  key.Binding
	Remove   key.Binding
	Create   key.Binding
	Back     key.Binding
	MainMenu key.Binding
}

// DefaultContainerKeyMap returns default container keybindings
func DefaultContainerKeyMap() ContainerKeyMap {
	return ContainerKeyMap{
		Refresh: key.NewBinding(
			key.WithKeys("r"),
			key.WithHelp("r", "refresh"),
		),
		Logs: key.NewBinding(
			key.WithKeys("l"),
			key.WithHelp("l", "logs"),
		),
		Stop: key.NewBinding(
			key.WithKeys("s"),
			key.WithHelp("s", "stop"),
		),
		Start: key.NewBinding(
			key.WithKeys("a"),
			key.WithHelp("a", "start"),
		),
		Restart: key.NewBinding(
			key.WithKeys("t"),
			key.WithHelp("t", "restart"),
		),
		Remove: key.NewBinding(
			key.WithKeys("x"),
			key.WithHelp("x", "remove"),
		),
		Create: key.NewBinding(
			key.WithKeys("c"),
			key.WithHelp("c", "create"),
		),
		Back: key.NewBinding(
			key.WithKeys("esc", "backspace"),
			key.WithHelp("esc", "back"),
		),
		MainMenu: key.NewBinding(
			key.WithKeys("m"),
			key.WithHelp("m", "main menu"),
		),
	}
}

// ContainerModel manages container view state
type ContainerModel struct {
	docker            *client.DockerClient
	containerList     list.Model
	selectedContainer *Summary
	keyMap            ContainerKeyMap
	state             string // "list", "logs", "confirm", "create"
	width             int
	height            int
	showAll           bool
	confirmMsg        string
	confirmAction     string
	viewport          viewport.Model // For logs and other scrollable content
	loading           bool
	error             error
	createModel       *ContainerCreateModel // Form for container creation
	spinner           spinner.Model
}

// NewContainerModel creates a new container model
func NewContainerModel(docker *client.DockerClient) *ContainerModel {
	keyMap := DefaultContainerKeyMap()

	// Set up container list with a more specific delegate
	delegate := list.NewDefaultDelegate()
	// Make sure titles and descriptions are set with proper styles
	delegate.Styles.NormalTitle = lipgloss.NewStyle().Foreground(ColorText).Bold(true)
	delegate.Styles.NormalDesc = lipgloss.NewStyle().Foreground(ColorSubtle)
	delegate.Styles.SelectedTitle = lipgloss.NewStyle().Foreground(ColorPrimary).Bold(true)
	delegate.Styles.SelectedDesc = lipgloss.NewStyle().Foreground(ColorText)

	// Ensure proper height for list items
	delegate.SetHeight(2) // Adjust if you need more space

	containerList := list.New([]list.Item{}, delegate, 0, 0)
	containerList.Title = "Containers"
	containerList.Styles.Title = StyleTitle
	containerList.SetShowStatusBar(true)    // Show status bar at the bottom
	containerList.SetFilteringEnabled(true) // Enable filtering
	containerList.AdditionalFullHelpKeys = func() []key.Binding {
		return []key.Binding{
			keyMap.Refresh,
			keyMap.Logs,
			keyMap.Stop,
			keyMap.Start,
			keyMap.Restart,
			keyMap.Remove,
			keyMap.Create,
			keyMap.Back,
			keyMap.MainMenu,
		}
	}

	// Set up viewport for logs
	vp := viewport.New(0, 0)
	vp.Style = lipgloss.NewStyle().
		BorderStyle(lipgloss.RoundedBorder()).
		BorderForeground(ColorPrimary)

	// Set up spinner
	s := spinner.New()
	s.Spinner = spinner.Dot
	s.Style = lipgloss.NewStyle().Foreground(ColorPrimary)

	return &ContainerModel{
		docker:        docker,
		containerList: containerList,
		keyMap:        keyMap,
		state:         "list",
		showAll:       true,
		viewport:      vp,
		loading:       true,
		spinner:       s,
	}
}

// Init initializes the model
func (m *ContainerModel) Init() tea.Cmd {
	// Initialize dimensions for the list view
	headerHeight := 6 // Adjust based on your layout
	footerHeight := 2
	listHeight := m.height - headerHeight - footerHeight
	if listHeight < 1 {
		listHeight = 10 // Fallback minimum
	}

	listWidth := m.width - 4
	if listWidth < 10 {
		listWidth = 40 // Fallback minimum
	}

	m.containerList.SetSize(listWidth, listHeight)

	// Update viewport dimensions for logs view
	m.viewport.Width = m.width - 4
	m.viewport.Height = m.height - headerHeight - footerHeight

	return tea.Batch(
		m.fetchContainers(),
		m.spinner.Tick,
	)
}

func convertPorts(ports []container.Port) []Port {
	newPorts := make([]Port, len(ports))
	for i, p := range ports {
		newPorts[i] = Port{
			IP:          p.IP,
			PrivatePort: p.PrivatePort,
			PublicPort:  p.PublicPort,
			Type:        p.Type,
		}
	}
	return newPorts
}

func convertMounts(mounts []container.MountPoint) []MountPoint {
	newMounts := make([]MountPoint, len(mounts))
	for i, m := range mounts {
		newMounts[i] = MountPoint{
			Type:        string(m.Type),
			Name:        m.Name,
			Source:      m.Source,
			Destination: m.Destination,
			Driver:      m.Driver,
			Mode:        m.Mode,
			RW:          m.RW,
			Propagation: string(m.Propagation),
		}
	}
	return newMounts
}

func containerSummaryToSummary(c container.Summary) Summary {
	var networkSettings *NetworkSettingsSummary
	if c.NetworkSettings != nil {
		networkSettings = &NetworkSettingsSummary{Networks: c.NetworkSettings.Networks}
	}

	return Summary{
		ID:         c.ID,
		Names:      c.Names,
		Image:      c.Image,
		ImageID:    c.ImageID,
		Command:    c.Command,
		Created:    c.Created,
		Ports:      convertPorts(c.Ports),
		SizeRw:     c.SizeRw,
		SizeRootFs: c.SizeRootFs,
		Labels:     c.Labels,
		State:      c.State,
		Status:     c.Status,
		HostConfig: struct {
			NetworkMode string            `json:"NetworkMode,omitempty"`
			Annotations map[string]string `json:"Annotations,omitempty"`
		}{
			NetworkMode: c.HostConfig.NetworkMode,
			Annotations: c.HostConfig.Annotations,
		},
		NetworkSettings: networkSettings,
		Mounts:          convertMounts(c.Mounts),
	}
}

// fetchContainers returns a command that fetches container data
func (m *ContainerModel) fetchContainers() tea.Cmd {
	return func() tea.Msg {
		ctx := context.Background()
		containers, err := m.docker.Client.ContainerList(ctx, container.ListOptions{All: m.showAll})
		summaries := make([]Summary, len(containers))
		for i, c := range containers {
			summaries[i] = containerSummaryToSummary(c)
		}
		return ContainerListMsg{
			Containers: summaries,
			Error:      err,
		}
	}
}

// fetchContainerLogs returns a command that fetches logs for a container
func (m *ContainerModel) fetchContainerLogs(containerID string) tea.Cmd {
	return func() tea.Msg {
		ctx := context.Background()
		options := container.LogsOptions{
			ShowStdout: true,
			ShowStderr: true,
			Tail:       "100",
		}

		// Get logs reader
		logsReader, err := m.docker.Client.ContainerLogs(ctx, containerID, options)
		if err != nil {
			return ContainerLogsMsg{Error: err}
		}
		defer logsReader.Close()

		// Read logs
		buf := new(bytes.Buffer)
		_, err = buf.ReadFrom(logsReader)
		if err != nil {
			return ContainerLogsMsg{Error: err}
		}

		return ContainerLogsMsg{
			Logs:  buf.String(),
			Error: nil,
		}
	}
}

// performContainerAction returns a command that performs an action on a container
func (m *ContainerModel) performContainerAction(action string, containerID string) tea.Cmd {
	return func() tea.Msg {
		ctx := context.Background()
		var err error

		switch action {
		case "stop":
			timeout := 10 // seconds
			err = m.docker.Client.ContainerStop(ctx, containerID, container.StopOptions{Timeout: &timeout})
		case "start":
			err = m.docker.Client.ContainerStart(ctx, containerID, container.StartOptions{})
		case "restart":
			timeout := 10 // seconds
			err = m.docker.Client.ContainerRestart(ctx, containerID, container.StopOptions{Timeout: &timeout})
		case "remove":
			err = m.docker.Client.ContainerRemove(ctx, containerID, container.RemoveOptions{Force: false})
		}

		return ContainerActionMsg{
			Action:      action,
			ContainerID: containerID,
			Error:       err,
		}
	}
}

// Update handles messages and updates the model
func (m *ContainerModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmds []tea.Cmd

	switch msg := msg.(type) {
	case tea.KeyMsg:
		// Handle container creation view separately
		if m.state == "create" {
			if msg.String() == "esc" {
				m.state = "list"
				return m, nil
			}

			var cmd tea.Cmd
			newModel, cmd := m.createModel.Update(msg)
			if updatedModel, ok := newModel.(*ContainerCreateModel); ok {
				m.createModel = updatedModel
			}
			return m, cmd
		}

		switch m.state {
		case "list":
			switch {
			case key.Matches(msg, m.keyMap.Back):
				// Only at the list level do we return to main menu
				return m, func() tea.Msg {
					return ReturnToMainMsg{}
				}

			case key.Matches(msg, m.keyMap.MainMenu):
				return m, func() tea.Msg {
					return ReturnToMainMsg{}
				}

			case key.Matches(msg, m.keyMap.Refresh):
				m.loading = true
				return m, tea.Batch(m.fetchContainers(), m.spinner.Tick)

			case key.Matches(msg, m.keyMap.Logs):
				if item, ok := m.containerList.SelectedItem().(ContainerItem); ok {
					m.selectedContainer = &item.container
					m.state = "logs"
					return m, m.fetchContainerLogs(item.container.ID)
				}

			case key.Matches(msg, m.keyMap.Stop):
				if item, ok := m.containerList.SelectedItem().(ContainerItem); ok {
					m.selectedContainer = &item.container
					m.confirmMsg = fmt.Sprintf("Are you sure you want to stop container %s?", strings.TrimPrefix(item.container.Names[0], "/"))
					m.confirmAction = "stop"
					m.state = "confirm"
					return m, nil
				}

			case key.Matches(msg, m.keyMap.Start):
				if item, ok := m.containerList.SelectedItem().(ContainerItem); ok {
					m.selectedContainer = &item.container
					m.confirmMsg = fmt.Sprintf("Are you sure you want to start container %s?", strings.TrimPrefix(item.container.Names[0], "/"))
					m.confirmAction = "start"
					m.state = "confirm"
					return m, nil
				}

			case key.Matches(msg, m.keyMap.Restart):
				if item, ok := m.containerList.SelectedItem().(ContainerItem); ok {
					m.selectedContainer = &item.container
					m.confirmMsg = fmt.Sprintf("Are you sure you want to restart container %s?", strings.TrimPrefix(item.container.Names[0], "/"))
					m.confirmAction = "restart"
					m.state = "confirm"
					return m, nil
				}

			case key.Matches(msg, m.keyMap.Remove):
				if item, ok := m.containerList.SelectedItem().(ContainerItem); ok {
					m.selectedContainer = &item.container
					m.confirmMsg = fmt.Sprintf("Are you sure you want to remove container %s?", strings.TrimPrefix(item.container.Names[0], "/"))
					m.confirmAction = "remove"
					m.state = "confirm"
					return m, nil
				}

			case key.Matches(msg, m.keyMap.Create):
				// Initialize container creation model
				m.createModel = NewContainerCreateModel(m.docker)
				m.createModel.width = m.width
				m.createModel.height = m.height
				m.state = "create"
				return m, m.createModel.Init()
			}

		case "logs":
			switch {
			case key.Matches(msg, m.keyMap.Back):
				m.state = "list"
				return m, nil
			default:
				// Handle viewport scrolling
				var cmd tea.Cmd
				m.viewport, cmd = m.viewport.Update(msg)
				return m, cmd
			}

		case "confirm":
			switch msg.String() {
			case "y", "Y":
				if m.selectedContainer != nil {
					return m, m.performContainerAction(m.confirmAction, m.selectedContainer.ID)
				}
				m.state = "list"
			case "n", "N", "esc":
				m.state = "list"
			}
			return m, nil
		}

	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height

		// Update list dimensions - be more specific with dimensions
		headerHeight := 6 // Adjust based on your layout
		footerHeight := 2
		listHeight := m.height - headerHeight - footerHeight
		if listHeight < 1 {
			listHeight = 10 // Fallback minimum
		}

		listWidth := m.width - 4
		if listWidth < 10 {
			listWidth = 40 // Fallback minimum
		}

		m.containerList.SetSize(listWidth, listHeight)

		// Update viewport dimensions
		m.viewport.Width = m.width - 4
		m.viewport.Height = m.height - headerHeight - footerHeight

		// Update create model dimensions if active
		if m.createModel != nil {
			m.createModel.width = msg.Width
			m.createModel.height = msg.Height
		}

		return m, nil

	case spinner.TickMsg:
		var cmd tea.Cmd
		m.spinner, cmd = m.spinner.Update(msg)
		cmds = append(cmds, cmd)

	case ContainerListMsg:
		m.loading = false
		if msg.Error != nil {
			m.error = msg.Error
			return m, nil
		}

		items := make([]list.Item, 0, len(msg.Containers))
		for _, c := range msg.Containers {
			name := strings.TrimPrefix(c.Names[0], "/")
			// Format created time
			createdTime := time.Unix(c.Created, 0)
			created := formatter.FormatTime(createdTime)

			// Include state in description using color formatting
			stateStyle := StyleTableRow
			switch c.State {
			case "running":
				stateStyle = StyleSuccess
			case "exited":
				stateStyle = StyleSubtle
			case "created":
				stateStyle = StyleWarning
			}

			status := stateStyle.Render(c.Status)

			desc := fmt.Sprintf("ID: %s • Image: %s • Created: %s • Status: %s",
				c.ID[:12],
				c.Image,
				created,
				status,
			)

			items = append(items, ContainerItem{
				container: c,
				title:     name,
				desc:      desc,
			})
		}

		cmd := m.containerList.SetItems(items)
		return m, cmd

	case ContainerLogsMsg:
		if msg.Error != nil {
			m.error = msg.Error
			m.state = "list"
			return m, nil
		}

		m.viewport.SetContent(msg.Logs)
		m.viewport.GotoTop()
		return m, nil

	case ContainerActionMsg:
		if msg.Error != nil {
			m.error = msg.Error
			m.state = "list"
			return m, nil
		}

		// Refresh container list after successful action
		m.loading = true
		m.state = "list"
		return m, m.fetchContainers()

	case ContainerCreateMsg:
		// Container was created, refresh the list
		m.loading = true
		m.state = "list"
		return m, m.fetchContainers()
	}

	// Update list in list state
	if m.state == "list" {
		var cmd tea.Cmd
		m.containerList, cmd = m.containerList.Update(msg)
		cmds = append(cmds, cmd)
	}

	// Update create model if it exists
	if m.state == "create" && m.createModel != nil {
		var cmd tea.Cmd
		newModel, cmd := m.createModel.Update(msg)
		if updatedModel, ok := newModel.(*ContainerCreateModel); ok {
			m.createModel = updatedModel
		}
		cmds = append(cmds, cmd)
	}

	return m, tea.Batch(cmds...)
}

// View renders the current view
func (m *ContainerModel) View() string {
	if m.loading {
		return StyleMainLayout.Render(
			lipgloss.JoinVertical(lipgloss.Center,
				StyleTitle.Render("Containers"),
				fmt.Sprintf("%s Loading containers...", m.spinner.View()),
			),
		)
	}

	if m.error != nil {
		errorBox := StyleInfoBox.
			BorderForeground(ColorError).
			Render(StyleError.Render(fmt.Sprintf("Error: %v", m.error)))

		help := "Press r to retry, esc to go back"
		return StyleMainLayout.Render(
			lipgloss.JoinVertical(lipgloss.Left,
				StyleTitle.Render("Container Management"),
				errorBox,
				help,
			),
		)
	}

	var content string
	switch m.state {
	case "list":
		content = lipgloss.JoinVertical(lipgloss.Left,
			StyleTitle.Render("Container Management"),
			"",
			m.containerList.View(),
		)

	case "logs":
		if m.selectedContainer != nil {
			name := strings.TrimPrefix(m.selectedContainer.Names[0], "/")
			title := fmt.Sprintf("Logs: %s", name)

			content = lipgloss.JoinVertical(lipgloss.Left,
				StyleTitle.Render(title),
				m.viewport.View(),
				StyleFooter.Render("Press esc to go back"),
			)
		}

	case "confirm":
		confirmBox := StyleInfoBox.Render(
			lipgloss.JoinVertical(lipgloss.Left,
				m.confirmMsg,
				"",
				"Press (y)es to confirm or (n)o to cancel",
			),
		)

		content = lipgloss.JoinVertical(lipgloss.Left,
			StyleTitle.Render("Confirm Action"),
			"",
			confirmBox,
		)

	case "create":
		if m.createModel != nil {
			return m.createModel.View()
		}
	}

	if m.state == "list" {
		helpText := StyleHelp.Render(
			"r: Refresh • l: Logs • s: Stop • a: Start • t: Restart • x: Remove • c: Create • m: Main menu",
		)
		content = lipgloss.JoinVertical(lipgloss.Left, content, "", helpText)
	}

	return StyleMainLayout.Render(content)
}
