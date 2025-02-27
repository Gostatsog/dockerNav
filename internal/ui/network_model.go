package ui

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/Gostatsog/dockerNav/internal/client"
	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/list"
	"github.com/charmbracelet/bubbles/spinner"
	"github.com/charmbracelet/bubbles/textinput"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/docker/docker/api/types/network"
)

// NetworkListMsg carries network data after fetching
type NetworkListMsg struct {
	Networks []network.Summary
	Error    error
}

// NetworkInspectMsg carries network inspection data
type NetworkInspectMsg struct {
	Network network.Inspect
	Error   error
}

// NetworkActionMsg carries results of network actions
type NetworkActionMsg struct {
	Action    string
	NetworkID string
	Error     error
}

// NetworkCreateMsg carries results of creating a network
type NetworkCreateMsg struct {
	Success bool
	ID      string
	Error   error
}

// NetworkItem represents a network in the list
type NetworkItem struct {
	network network.Summary
	title   string
	desc    string
}

// FilterValue implements list.Item interface
func (i NetworkItem) FilterValue() string { return i.title }

// Title returns the title for the list item
func (i NetworkItem) Title() string { return i.title }

// Description returns the description for the list item
func (i NetworkItem) Description() string { return i.desc }

// NetworkKeyMap defines keybindings for network operations
type NetworkKeyMap struct {
	Refresh  key.Binding
	Inspect  key.Binding
	Create   key.Binding
	Remove   key.Binding
	Back     key.Binding
}

// DefaultNetworkKeyMap returns default network keybindings
func DefaultNetworkKeyMap() NetworkKeyMap {
	return NetworkKeyMap{
		Refresh: key.NewBinding(
			key.WithKeys("r"),
			key.WithHelp("r", "refresh"),
		),
		Inspect: key.NewBinding(
			key.WithKeys("i"),
			key.WithHelp("i", "inspect"),
		),
		Create: key.NewBinding(
			key.WithKeys("c"),
			key.WithHelp("c", "create"),
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

// NetworkModel manages network view state
type NetworkModel struct {
	docker        *client.DockerClient
	networkList   list.Model
	keyMap        NetworkKeyMap
	state         string // "list", "inspect", "create", "confirm"
	width         int
	height        int
	spin          spinner.Model
	viewport      viewport.Model
	textInputs    []textinput.Model
	selectedNetwork *network.Summary
	inspectedNetwork *network.Inspect
	confirmMsg      string
	confirmAction   string
	loading       bool
	focusIndex    int // For managing focus between input fields
	error         error
}

// NewNetworkModel creates a new network model
func NewNetworkModel(docker *client.DockerClient) *NetworkModel {
	keyMap := DefaultNetworkKeyMap()
	
	// Set up network list with improved delegate styling
	delegate := list.NewDefaultDelegate()
	delegate.Styles.NormalTitle = lipgloss.NewStyle().Foreground(ColorText).Bold(true)
	delegate.Styles.NormalDesc = lipgloss.NewStyle().Foreground(ColorSubtle)
	delegate.Styles.SelectedTitle = lipgloss.NewStyle().Foreground(ColorPrimary).Bold(true)
	delegate.Styles.SelectedDesc = lipgloss.NewStyle().Foreground(ColorText)
	delegate.SetHeight(2) // Adjust if needed
	
	networkList := list.New([]list.Item{}, delegate, 0, 0)
	networkList.Title = "Networks"
	networkList.Styles.Title = StyleTitle
	networkList.SetShowStatusBar(true)
	networkList.SetFilteringEnabled(true)
	networkList.AdditionalFullHelpKeys = func() []key.Binding {
		return []key.Binding{
			keyMap.Refresh,
			keyMap.Inspect,
			keyMap.Create,
			keyMap.Remove,
			keyMap.Back,
		}
	}

	// Set up spinner for loading states
	s := spinner.New()
	s.Spinner = spinner.Dot
	s.Style = lipgloss.NewStyle().Foreground(ColorPrimary)

	// Set up text inputs for creating networks
	inputs := make([]textinput.Model, 2)
	for i := range inputs {
		inputs[i] = textinput.New()
		inputs[i].Width = 40
	}
	
	inputs[0].Placeholder = "Network Name"
	inputs[0].Focus()
	inputs[0].PromptStyle = lipgloss.NewStyle().Foreground(ColorPrimary)
	inputs[0].TextStyle = lipgloss.NewStyle().Foreground(ColorText)
	
	inputs[1].Placeholder = "Driver (bridge, overlay, etc.)"
	inputs[1].PromptStyle = lipgloss.NewStyle().Foreground(ColorPrimary)
	inputs[1].TextStyle = lipgloss.NewStyle().Foreground(ColorText)

	// Set up viewport for scrollable content
	vp := viewport.New(0, 0)
	vp.Style = lipgloss.NewStyle().
		BorderStyle(lipgloss.RoundedBorder()).
		BorderForeground(ColorPrimary)

	return &NetworkModel{
		docker:      docker,
		networkList: networkList,
		keyMap:      keyMap,
		state:       "list",
		spin:        s,
		viewport:    vp,
		textInputs:  inputs,
		loading:     true,
	}
}

// Init initializes the model
func (m *NetworkModel) Init() tea.Cmd {
    // Initialize dimensions for the list view
    headerHeight := 6
    footerHeight := 2
    listHeight := m.height - headerHeight - footerHeight
    if listHeight < 1 {
        listHeight = 10 // Minimum height
    }
    
    listWidth := m.width - 4
    if listWidth < 10 {
        listWidth = 40 // Minimum width
    }
    
    m.networkList.SetSize(listWidth, listHeight)
    
    // Update viewport dimensions
    m.viewport.Width = m.width - 4
    m.viewport.Height = m.height - headerHeight - footerHeight
    
    return tea.Batch(m.fetchNetworks(), m.spin.Tick)
}

// fetchNetworks returns a command that fetches network data
func (m *NetworkModel) fetchNetworks() tea.Cmd {
	return func() tea.Msg {
		ctx := context.Background()
		networks, err := m.docker.Client.NetworkList(ctx, network.ListOptions{})
		return NetworkListMsg{
			Networks: networks,
			Error:    err,
		}
	}
}

// inspectNetwork returns a command that inspects a network
func (m *NetworkModel) inspectNetwork(networkID string) tea.Cmd {
	return func() tea.Msg {
		ctx := context.Background()
		networkDetails, err := m.docker.Client.NetworkInspect(ctx, networkID, network.InspectOptions{})
		return NetworkInspectMsg{
			Network: networkDetails,
			Error:   err,
		}
	}
}

// createNetwork returns a command that creates a network
func (m *NetworkModel) createNetwork(name, driver string) tea.Cmd {
	return func() tea.Msg {
		ctx := context.Background()
		
		options := network.CreateOptions{
			Driver: driver,
		}
		
		// Set default driver if not specified
		if driver == "" {
			options.Driver = "bridge"
		}
		
		response, err := m.docker.Client.NetworkCreate(ctx, name, options)
		
		if err != nil {
			return NetworkCreateMsg{Success: false, Error: err}
		}
		
		return NetworkCreateMsg{Success: true, ID: response.ID, Error: nil}
	}
}

// performNetworkAction returns a command that performs an action on a network
func (m *NetworkModel) performNetworkAction(action string, networkID string) tea.Cmd {
	return func() tea.Msg {
		ctx := context.Background()
		var err error
		
		switch action {
		case "remove":
			err = m.docker.Client.NetworkRemove(ctx, networkID)
		}
		
		return NetworkActionMsg{
			Action:    action,
			NetworkID: networkID,
			Error:     err,
		}
	}
}

// updateInputs updates the focus of text inputs
func (m *NetworkModel) updateInputs(msg tea.Msg) tea.Cmd {
	cmds := make([]tea.Cmd, len(m.textInputs))
	
	// Update all text inputs
	for i := range m.textInputs {
		m.textInputs[i].PromptStyle = lipgloss.NewStyle().Foreground(ColorSubtle)
		m.textInputs[i].TextStyle = lipgloss.NewStyle().Foreground(ColorText)
	}
	
	// Set focus on the current input
	if m.focusIndex < len(m.textInputs) {
		m.textInputs[m.focusIndex].PromptStyle = lipgloss.NewStyle().Foreground(ColorPrimary)
		m.textInputs[m.focusIndex].TextStyle = lipgloss.NewStyle().Foreground(ColorPrimary)
		
		// Only update the focused input
		var cmd tea.Cmd
		m.textInputs[m.focusIndex], cmd = m.textInputs[m.focusIndex].Update(msg)
		cmds[m.focusIndex] = cmd
	}
	
	return tea.Batch(cmds...)
}

// Update handles messages and updates the model
func (m *NetworkModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmds []tea.Cmd

	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch m.state {
		case "list":
			switch {
			case key.Matches(msg, m.keyMap.Refresh):
				m.loading = true
				return m, tea.Batch(m.fetchNetworks(), m.spin.Tick)
				
			case key.Matches(msg, m.keyMap.Inspect):
				if item, ok := m.networkList.SelectedItem().(NetworkItem); ok {
					m.selectedNetwork = &item.network
					m.state = "inspect"
					m.loading = true
					return m, tea.Batch(m.inspectNetwork(item.network.ID), m.spin.Tick)
				}
				
			case key.Matches(msg, m.keyMap.Create):
				m.state = "create"
				m.focusIndex = 0
				m.textInputs[0].Focus()
				m.textInputs[1].Blur()
				return m, nil
				
			case key.Matches(msg, m.keyMap.Remove):
				if item, ok := m.networkList.SelectedItem().(NetworkItem); ok {
					m.selectedNetwork = &item.network
					m.confirmMsg = fmt.Sprintf("Are you sure you want to remove network %s?", item.network.Name)
					m.confirmAction = "remove"
					m.state = "confirm"
					return m, nil
				}
			}
			
		case "inspect":
			switch {
			case key.Matches(msg, m.keyMap.Back):
				m.state = "list"
				m.inspectedNetwork = nil
				return m, nil
			default:
				// Handle viewport scrolling
				var cmd tea.Cmd
				m.viewport, cmd = m.viewport.Update(msg)
				return m, cmd
			}
			
		case "create":
			switch msg.String() {
			case "tab", "shift+tab":
				// Cycle between inputs
				if msg.String() == "tab" {
					m.focusIndex = (m.focusIndex + 1) % len(m.textInputs)
				} else {
					m.focusIndex = (m.focusIndex - 1 + len(m.textInputs)) % len(m.textInputs)
				}
				
				// Update input focus
				for i := range m.textInputs {
					if i == m.focusIndex {
						m.textInputs[i].Focus()
					} else {
						m.textInputs[i].Blur()
					}
				}
				
				return m, nil
				
			case "enter":
				// Create the network
				networkName := m.textInputs[0].Value()
				driver := m.textInputs[1].Value()
				
				if networkName == "" {
					return m, nil
				}
				
				m.loading = true
				m.state = "list"
				
				// Reset inputs
				for i := range m.textInputs {
					m.textInputs[i].Reset()
				}
				
				return m, tea.Batch(m.createNetwork(networkName, driver), m.spin.Tick)
				
			case "esc":
				m.state = "list"
				// Reset inputs
				for i := range m.textInputs {
					m.textInputs[i].Reset()
				}
				return m, nil
				
			default:
				// Update the inputs
				return m, m.updateInputs(msg)
			}
			
		case "confirm":
			switch msg.String() {
			case "y", "Y":
				if m.selectedNetwork != nil {
					return m, m.performNetworkAction(m.confirmAction, m.selectedNetwork.ID)
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
		
		// Update list dimensions with better constraints
		headerHeight := 6
		footerHeight := 2
		listHeight := m.height - headerHeight - footerHeight
		if listHeight < 1 {
			listHeight = 10 // Minimum height
		}
		
		listWidth := m.width - 4
		if listWidth < 10 {
			listWidth = 40 // Minimum width
		}
		
		m.networkList.SetSize(listWidth, listHeight)
		
		// Update viewport dimensions
		m.viewport.Width = m.width - 4
		m.viewport.Height = m.height - headerHeight - footerHeight
		
		return m, nil

	case spinner.TickMsg:
		var cmd tea.Cmd
		m.spin, cmd = m.spin.Update(msg)
		cmds = append(cmds, cmd)

	case NetworkListMsg:
		m.loading = false
		if msg.Error != nil {
			m.error = msg.Error
			return m, nil
		}

		items := make([]list.Item, 0, len(msg.Networks))
		for _, net := range msg.Networks {
			desc := fmt.Sprintf("ID: %s • Driver: %s • Scope: %s",
				net.ID[:12],
				net.Driver,
				net.Scope,
			)
			
			items = append(items, NetworkItem{
				network: net,
				title:   net.Name,
				desc:    desc,
			})
		}
		
		cmd := m.networkList.SetItems(items)
		return m, cmd
		
	case NetworkInspectMsg:
		m.loading = false
		if msg.Error != nil {
			m.error = msg.Error
			m.state = "list"
			return m, nil
		}
		
		m.inspectedNetwork = &msg.Network
		
		// Format network details for display
		var detailsBuilder strings.Builder
		detailsBuilder.WriteString(fmt.Sprintf("Name: %s\n", msg.Network.Name))
		detailsBuilder.WriteString(fmt.Sprintf("ID: %s\n", msg.Network.ID))
		detailsBuilder.WriteString(fmt.Sprintf("Created: %s\n", msg.Network.Created.Format(time.RFC3339)))
		detailsBuilder.WriteString(fmt.Sprintf("Driver: %s\n", msg.Network.Driver))
		detailsBuilder.WriteString(fmt.Sprintf("Scope: %s\n", msg.Network.Scope))
		detailsBuilder.WriteString(fmt.Sprintf("Internal: %v\n", msg.Network.Internal))
		detailsBuilder.WriteString(fmt.Sprintf("Enable IPv6: %v\n", msg.Network.EnableIPv6))
		detailsBuilder.WriteString(fmt.Sprintf("Attachable: %v\n", msg.Network.Attachable))
		detailsBuilder.WriteString(fmt.Sprintf("Ingress: %v\n", msg.Network.Ingress))
		
		// IPAM Configuration
		detailsBuilder.WriteString("\nIPAM Configuration:\n")
		detailsBuilder.WriteString(fmt.Sprintf("  Driver: %s\n", msg.Network.IPAM.Driver))
		
		for i, conf := range msg.Network.IPAM.Config {
			detailsBuilder.WriteString(fmt.Sprintf("  Config %d:\n", i+1))
			if conf.Subnet != "" {
				detailsBuilder.WriteString(fmt.Sprintf("    Subnet: %s\n", conf.Subnet))
			}
			if conf.Gateway != "" {
				detailsBuilder.WriteString(fmt.Sprintf("    Gateway: %s\n", conf.Gateway))
			}
			if conf.IPRange != "" {
				detailsBuilder.WriteString(fmt.Sprintf("    IP Range: %s\n", conf.IPRange))
			}
		}
		
		// Connected Containers
		detailsBuilder.WriteString("\nConnected Containers:\n")
		if len(msg.Network.Containers) > 0 {
			for id, endpoint := range msg.Network.Containers {
				detailsBuilder.WriteString(fmt.Sprintf("  Container: %s\n", endpoint.Name))
				detailsBuilder.WriteString(fmt.Sprintf("    ID: %s\n", id[:12]))
				detailsBuilder.WriteString(fmt.Sprintf("    IPv4 Address: %s\n", endpoint.IPv4Address))
				if endpoint.IPv6Address != "" {
					detailsBuilder.WriteString(fmt.Sprintf("    IPv6 Address: %s\n", endpoint.IPv6Address))
				}
				detailsBuilder.WriteString(fmt.Sprintf("    MAC Address: %s\n", endpoint.MacAddress))
			}
		} else {
			detailsBuilder.WriteString("  No containers connected\n")
		}
		
		m.viewport.SetContent(detailsBuilder.String())
		m.viewport.GotoTop()
		return m, nil
		
	case NetworkCreateMsg:
		m.loading = false
		if msg.Error != nil {
			m.error = msg.Error
			return m, nil
		}
		
		// Refresh network list after successful creation
		return m, m.fetchNetworks()
		
	case NetworkActionMsg:
		m.loading = false
		if msg.Error != nil {
			m.error = msg.Error
			return m, nil
		}
		
		// Refresh network list after successful action
		m.state = "list"
		return m, m.fetchNetworks()
	}

	// Update list in list state
	if m.state == "list" {
		var cmd tea.Cmd
		m.networkList, cmd = m.networkList.Update(msg)
		cmds = append(cmds, cmd)
	}

	return m, tea.Batch(cmds...)
}

// View renders the current view
func (m *NetworkModel) View() string {
	if m.loading {
		return StyleMainLayout.Render(
			lipgloss.JoinVertical(lipgloss.Center,
				StyleTitle.Render("Networks"),
				fmt.Sprintf("%s Loading networks...", m.spin.View()),
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
				StyleTitle.Render("Network Management"),
				errorBox, 
				help,
			),
		)
	}

	var content string
	switch m.state {
	case "list":
		// Check if list view is empty
		listView := m.networkList.View()
		if strings.TrimSpace(listView) == "" {
			content = lipgloss.JoinVertical(lipgloss.Left,
				StyleTitle.Render("Network Management"),
				"",
				StyleInfoBox.Render("No networks to display or list rendering issue."),
				StyleFooter.Render("Press r to refresh, c to create a new network"),
			)
		} else {
			content = lipgloss.JoinVertical(lipgloss.Left,
				StyleTitle.Render("Network Management"),
				"",
				listView,
			)
		}
		
	case "inspect":
		if m.inspectedNetwork != nil {
			title := fmt.Sprintf("Network Details: %s", m.inspectedNetwork.Name)
			
			content = lipgloss.JoinVertical(lipgloss.Left,
				StyleTitle.Render(title),
				m.viewport.View(),
				StyleFooter.Render("Press esc to go back"),
			)
		}
		
	case "create":
		inputBox := StyleInfoBox.Render(
			lipgloss.JoinVertical(lipgloss.Left,
				"Create New Network:",
				"",
				fmt.Sprintf("%s %s", m.textInputs[0].PromptStyle.Render("Name:"), m.textInputs[0].View()),
				fmt.Sprintf("%s %s", m.textInputs[1].PromptStyle.Render("Driver:"), m.textInputs[1].View()),
				"",
				"Tab: Next Field • Enter: Create • Esc: Cancel",
			),
		)
		
		content = lipgloss.JoinVertical(lipgloss.Left,
			StyleTitle.Render("Create Network"),
			"",
			inputBox,
		)
		
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

	if m.state == "list" {
		helpText := StyleHelp.Render(
			"r: Refresh • i: Inspect • c: Create • x: Remove • esc: Back",
		)
		content = lipgloss.JoinVertical(lipgloss.Left, content, "", helpText)
	}


	return StyleMainLayout.Render(content)
}