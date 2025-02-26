package ui

import (
	"fmt"
	"strings"

	"github.com/Gostatsog/dockerNav/internal/client"
	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/list"
	"github.com/charmbracelet/bubbles/spinner"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/docker/docker/api/types/volume"
)

// VolumeModel manages the volume view
type VolumeModel struct {
	dockerClient *client.DockerClient
	volumes      []volume.Volume
	volumeList   list.Model // Add a list model for consistent UI
	selected     int
	width        int
	height       int
	loading      bool
	error        error
	keyMap       VolumeKeyMap
	state        string // "list", "inspect", "confirm", "create"
	spinner      spinner.Model
}

// VolumeKeyMap defines keybindings for volume operations
type VolumeKeyMap struct {
	Refresh  key.Binding
	Inspect  key.Binding
	Create   key.Binding
	Remove   key.Binding
	Back     key.Binding
}

// VolumeListMsg contains the list of volumes
type VolumeListMsg struct {
	Volumes []volume.Volume
	Error   error
}

// VolumeItem represents a volume in the list
type VolumeItem struct {
	title  string
	desc   string
}

// FilterValue implements list.Item interface
func (i VolumeItem) FilterValue() string { return i.title }

// Title returns the title for the list item
func (i VolumeItem) Title() string { return i.title }

// Description returns the description for the list item
func (i VolumeItem) Description() string { return i.desc }

// NewVolumeModel creates a new volume model
func NewVolumeModel(dockerClient *client.DockerClient) *VolumeModel {
	keyMap := DefaultVolumeKeyMap()
	
	// Set up volume list with proper delegate styling
	delegate := list.NewDefaultDelegate()
	delegate.Styles.NormalTitle = lipgloss.NewStyle().Foreground(ColorText).Bold(true)
	delegate.Styles.NormalDesc = lipgloss.NewStyle().Foreground(ColorSubtle)
	delegate.Styles.SelectedTitle = lipgloss.NewStyle().Foreground(ColorPrimary).Bold(true)
	delegate.Styles.SelectedDesc = lipgloss.NewStyle().Foreground(ColorText)
	delegate.SetHeight(2) // Adjust if needed
	
	volumeList := list.New([]list.Item{}, delegate, 0, 0)
	volumeList.Title = "Volumes"
	volumeList.Styles.Title = StyleTitle
	volumeList.SetShowStatusBar(true)
	volumeList.SetFilteringEnabled(true)
	volumeList.AdditionalFullHelpKeys = func() []key.Binding {
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

	return &VolumeModel{
		dockerClient: dockerClient,
		volumes:      []volume.Volume{},
		volumeList:   volumeList,
		selected:     0,
		loading:      true,
		keyMap:       keyMap,
		state:        "list",
		spinner:      s,
	}
}


// Init initializes the model
func (m *VolumeModel) Init() tea.Cmd {
	return tea.Batch(m.fetchVolumes(), m.spinner.Tick)
}

// DefaultVolumeKeyMap returns default volume keybindings
func DefaultVolumeKeyMap() VolumeKeyMap {
	return VolumeKeyMap{
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

// fetchVolumes returns a command that fetches volumes
func (m *VolumeModel) fetchVolumes() tea.Cmd {
	return func() tea.Msg {
		ctx := m.dockerClient.Ctx
		volumes, err := m.dockerClient.Client.VolumeList(ctx, volume.ListOptions{})
		if err != nil {
			return VolumeListMsg{
				Volumes: []volume.Volume{}, // Ensure an empty slice is returned on error
				Error:   err,
			}
		}

		// Convert []*volume.Volume to []volume.Volume
		volumesList := make([]volume.Volume, len(volumes.Volumes))
		for i, v := range volumes.Volumes {
			if v != nil {
				volumesList[i] = *v // Dereference the pointer
			}
		}

		return VolumeListMsg{
			Volumes: volumesList,
			Error:   nil,
		}
	}
}

// Update handles messages and updates the model
func (m *VolumeModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
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
		
		m.volumeList.SetSize(listWidth, listHeight)
		return m, nil

	case tea.KeyMsg:
		switch msg.String() {
		case "j", "down":
			if m.selected < len(m.volumes)-1 {
				m.selected++
			}
		case "k", "up":
			if m.selected > 0 {
				m.selected--
			}
		case "r":
			// Refresh the volumes list
			return m, m.fetchVolumes()
		case "d":
			// Delete the selected volume
			if len(m.volumes) > 0 {
				return m, m.removeVolume(m.volumes[m.selected].Name)
			}
		case "c":
			// Create a new volume
			// TODO: Implement a form to create a new volume
			return m, m.fetchVolumes()
		case "i":
			// Inspect the selected volume
			if len(m.volumes) > 0 {
				return m, m.inspectVolume(m.volumes[m.selected].Name)
			}
		}

	case VolumeListMsg:
		m.loading = false
		if msg.Error != nil {
			m.error = msg.Error
			return m, nil
		}
		m.volumes = msg.Volumes
		if m.selected >= len(m.volumes) && len(m.volumes) > 0 {
			m.selected = len(m.volumes) - 1
		}
	case spinner.TickMsg:
		var cmd tea.Cmd
		m.spinner, cmd = m.spinner.Update(msg)
		return m, cmd
	
	}

	return m, nil
}

// removeVolume returns a command that removes a volume
func (m *VolumeModel) removeVolume(name string) tea.Cmd {
	return func() tea.Msg {
		ctx := m.dockerClient.Ctx
		err := m.dockerClient.Client.VolumeRemove(ctx, name, false)
		if err != nil {
			return VolumeListMsg{
				Volumes: m.volumes,
				Error:   err,
			}
		}
		// Refresh the list after removal
		return m.fetchVolumes()()
	}
}

// inspectVolume returns a command that inspects a volume
func (m *VolumeModel) inspectVolume(name string) tea.Cmd {
	return func() tea.Msg {
		ctx := m.dockerClient.Ctx
		vol, err := m.dockerClient.Client.VolumeInspect(ctx, name)
		if err != nil {
			return VolumeListMsg{
				Volumes: m.volumes,
				Error:   err,
			}
		}
		
		// For now, just refresh the volumes list
		// In a more complete implementation, this would show detailed info
		// in a separate view or modal
		fmt.Printf("Volume: %s\nDriver: %s\nMountpoint: %s\n", 
			vol.Name, vol.Driver, vol.Mountpoint)
		
		return m.fetchVolumes()()
	}
}

// View renders the model
func (m *VolumeModel) View() string {
	if m.loading {
		return StyleMainLayout.Render("Loading volumes...")
	}

	if m.error != nil {
		errorBox := StyleInfoBox.Copy().
			BorderForeground(ColorError).
			Render(StyleError.Render("Error loading volumes: " + m.error.Error()))
		
		helpText := "\nPress 'esc' to go back to the main menu."
		
		return StyleMainLayout.Render(
			lipgloss.JoinVertical(lipgloss.Center,
				StyleTitle.Render("Volume Management Error"),
				errorBox,
				helpText,
			),
		)
	}

	title := StyleTitle.Render("Volume Management")

	// Build volume list
	var volumeList string
	if len(m.volumes) == 0 {
		volumeList = "No volumes found"
	} else {
		// Table header
		header := lipgloss.JoinHorizontal(lipgloss.Left,
			StyleTableHeader.Width(25).Render("NAME"),
			StyleTableHeader.Width(15).Render("DRIVER"),
			StyleTableHeader.Width(30).Render("MOUNTPOINT"),
		)

		// Table rows
		rows := []string{header}
		for i, vol := range m.volumes {
			// For mountpoint, truncate if too long
			mountpoint := vol.Mountpoint
			if len(mountpoint) > 29 {
				mountpoint = mountpoint[:26] + "..."
			}

			// Style to use based on selection
			style := StyleTableRow
			if i == m.selected {
				style = style.Copy().Bold(true).Foreground(ColorPrimary)
			}

			row := lipgloss.JoinHorizontal(lipgloss.Left,
				style.Width(25).Render(vol.Name),
				style.Width(15).Render(vol.Driver),
				style.Width(30).Render(mountpoint),
			)
			rows = append(rows, row)
		}
		volumeList = strings.Join(rows, "\n")
	}

	// Create help text
	helpText := StyleHelp.Render(
		"↑/↓: Navigate • r: Refresh • d: Delete • c: Create • i: Inspect • esc: Back",
	)

	// Join everything together
	content := lipgloss.JoinVertical(lipgloss.Left,
		title,
		"",
		volumeList,
		"",
		helpText,
	)

	return StyleMainLayout.Render(content)
}