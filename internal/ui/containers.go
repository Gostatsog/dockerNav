package ui

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/Gostatsog/dockerNav/internal/client"
	"github.com/Gostatsog/dockerNav/pkg/formatter"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/list"
	"github.com/charmbracelet/bubbles/viewport"
	"github.com/charmbracelet/lipgloss"
	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/container"
)

// ContainerListMsg carries container data after fetching
type ContainerListMsg struct {
	Containers []container.Summary
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
	container container.Summary
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
	Refresh key.Binding
	Logs    key.Binding
	Stop    key.Binding
	Start   key.Binding
	Restart key.Binding
	Remove  key.Binding
	Back    key.Binding
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
		Back: key.NewBinding(
			key.WithKeys("esc", "backspace"),
			key.WithHelp("esc", "back"),
		),
	}
}

// ContainerModel manages container view state
type ContainerModel struct {
	docker       *client.DockerClient
	containerList list.Model
	selectedContainer *container.Summary
	keyMap       ContainerKeyMap
	state        string // "list", "logs", "confirm"
	width        int
	height       int
	showAll      bool
	confirmMsg   string
	confirmAction string
	viewport     viewport.Model // For logs and other scrollable content
	loading      bool
	error        error
}

// NewContainerModel creates a new container model
func NewContainerModel(docker *client.DockerClient) *ContainerModel {
	keyMap := DefaultContainerKeyMap()
	
	// Set up container list
	delegate := list.NewDefaultDelegate()
	containerList := list.New([]list.Item{}, delegate, 0, 0)
	containerList.Title = "Containers"
	containerList.Styles.Title = StyleTitle
	containerList.AdditionalFullHelpKeys = func() []key.Binding {
		return []key.Binding{
			keyMap.Refresh,
			keyMap.Logs,
			keyMap.Stop,
			keyMap.Start,
			keyMap.Restart,
			keyMap.Remove,
			keyMap.Back,
		}
	}

	// Set up viewport for logs
	vp := viewport.New(0, 0)
	vp.Style = lipgloss.NewStyle().
		BorderStyle(lipgloss.RoundedBorder()).
		BorderForeground(ColorPrimary)

	return &ContainerModel{
		docker:       docker,
		containerList: containerList,
		keyMap:       keyMap,
		state:        "list",
		showAll:      true,
		viewport:     vp,
		loading:      true,
	}
}

// Init initializes the model
func (m *ContainerModel) Init() tea.Cmd {
	return m.fetchContainers()
}

// fetchContainers returns a command that fetches container data
func (m *ContainerModel) fetchContainers() tea.Cmd {
	return func() tea.Msg {
		ctx := context.Background()
		containers, err := m.docker.Client.ContainerList(ctx, container.ListOptions{All: m.showAll})
		return ContainerListMsg{
			Containers: containers,
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
		buf := new(strings.Builder)
		_, err = buf.ReadFrom(logsReader)
		if err != nil {
			return ContainerLogsMsg{Error: err}
		}
		
		return ContainerLogsMsg{
			Logs: buf.String(),
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
			Action: action,
			ContainerID: containerID,
			Error:  err,
		}
	}
}

// Update handles messages and updates the model
func (m *ContainerModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmds []tea.Cmd

	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch m.state {
		case "list":
			switch {
			case key.Matches(msg, m.keyMap.Refresh):
				m.loading = true
				return m, m.fetchContainers()
				
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
		
		// Update list dimensions
		headerHeight := 6 // Adjust based on your layout
		footerHeight := 2
		m.containerList.SetSize(msg.Width-4, msg.Height-headerHeight-footerHeight)
		
		// Update viewport dimensions
		m.viewport.Width = msg.Width - 4
		m.viewport.Height = msg.Height - headerHeight - footerHeight

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
			
			desc := fmt.Sprintf("ID: %s • Image: %s • Created: %s • Status: %s",
				c.ID[:12],
				c.Image,
				created,
				c.Status,
			)
			
			items = append(items, ContainerItem{
				container: c,
				title:     name,
				desc:      desc,
			})
		}
		
		var cmd tea.Cmd
		m.containerList.SetItems(items)
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
	}

	// Update list in list state
	if m.state == "list" {
		var cmd tea.Cmd
		m.containerList, cmd = m.containerList.Update(msg)
		cmds = append(cmds, cmd)
	}

	return m, tea.Batch(cmds...)
}

// View renders the current view
func (m *ContainerModel) View() string {
	if m.loading {
		return StyleMainLayout.Render("Loading containers...")
	}

	if m.error != nil {
		errorBox := StyleInfoBox.Copy().
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
	}

	return StyleMainLayout.Render(content)
}

// Helper functions for formatting
func formatPorts(ports []types.Port) string {
	if len(ports) == 0 {
		return ""
	}

	var portStrings []string
	for _, port := range ports {
		if port.PublicPort != 0 {
			portStr := fmt.Sprintf("%d:%d/%s", port.PublicPort, port.PrivatePort, port.Type)
			portStrings = append(portStrings, portStr)
		} else {
			portStr := fmt.Sprintf("%d/%s", port.PrivatePort, port.Type)
			portStrings = append(portStrings, portStr)
		}
	}

	return strings.Join(portStrings, ", ")
}