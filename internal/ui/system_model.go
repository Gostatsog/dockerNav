package ui

import (
	"fmt"
	"strings"
	"time"

	"github.com/Gostatsog/dockerNav/internal/client"
	"github.com/Gostatsog/dockerNav/pkg/formatter"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/filters"
	"github.com/docker/docker/api/types/system"
)

// SystemModel manages the system view
type SystemModel struct {
	dockerClient *client.DockerClient
	info         system.Info // Use system.Info instead of types.Info
	version      types.Version
	diskUsage    types.DiskUsage
	width        int
	height       int
	loading      bool
	error        error
	selected     int
	subView      int // 0 = main, 1 = disk usage, 2 = version, 3 = info
}

// SystemInfoMsg contains system information
type SystemInfoMsg struct {
	Info    system.Info // Update this too
	Version types.Version
	Error   error
}

// SystemDiskUsageMsg contains disk usage information
type SystemDiskUsageMsg struct {
	DiskUsage types.DiskUsage
	Error     error
}

// NewSystemModel creates a new system model
func NewSystemModel(dockerClient *client.DockerClient) *SystemModel {
	return &SystemModel{
		dockerClient: dockerClient,
		loading:      true,
		selected:     0,
		subView:      0,
	}
}

// Init initializes the model
func (m *SystemModel) Init() tea.Cmd {
	return tea.Batch(
		m.fetchSystemInfo(),
		m.fetchDiskUsage(),
	)
}

// fetchSystemInfo returns a command that fetches system info
func (m *SystemModel) fetchSystemInfo() tea.Cmd {
	return func() tea.Msg {
		ctx := m.dockerClient.Ctx
		info, err := m.dockerClient.Client.Info(ctx) // No change needed here
		if err != nil {
			return SystemInfoMsg{
				Info:    system.Info{}, // Update to system.Info{}
				Version: types.Version{},
				Error:   err,
			}
		}

		version, err := m.dockerClient.Client.ServerVersion(ctx)
		if err != nil {
			return SystemInfoMsg{
				Info:    info,
				Version: types.Version{},
				Error:   err,
			}
		}

		return SystemInfoMsg{
			Info:    info,
			Version: version,
			Error:   nil,
		}
	}
}

// fetchDiskUsage returns a command that fetches disk usage
func (m *SystemModel) fetchDiskUsage() tea.Cmd {
	return func() tea.Msg {
		ctx := m.dockerClient.Ctx
		usage, err := m.dockerClient.Client.DiskUsage(ctx, types.DiskUsageOptions{})
		if err != nil {
			return SystemDiskUsageMsg{
				DiskUsage: types.DiskUsage{},
				Error:     err,
			}
		}

		return SystemDiskUsageMsg{
			DiskUsage: usage,
			Error:     nil,
		}
	}
}

// pruneSystem returns a command that prunes the system
func (m *SystemModel) pruneSystem() tea.Cmd {
	return func() tea.Msg {
		ctx := m.dockerClient.Ctx
		cli := m.dockerClient.Client

		// Prune containers
		_, err := cli.ContainersPrune(ctx, filters.NewArgs())
		if err != nil {
			return SystemInfoMsg{Error: err}
		}

		// Prune images
		_, err = cli.ImagesPrune(ctx, filters.NewArgs())
		if err != nil {
			return SystemInfoMsg{Error: err}
		}

		// Prune networks
		_, err = cli.NetworksPrune(ctx, filters.NewArgs())
		if err != nil {
			return SystemInfoMsg{Error: err}
		}

		// Prune volumes
		_, err = cli.VolumesPrune(ctx, filters.NewArgs())
		if err != nil {
			return SystemInfoMsg{Error: err}
		}

		// Refresh disk usage after pruning
		return m.fetchDiskUsage()()
	}
}

// Update handles messages and updates the model
func (m *SystemModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "1":
			m.subView = 0 // Main system view
		case "2":
			m.subView = 1 // Disk usage view
		case "3":
			m.subView = 2 // Version details view
		case "4":
			m.subView = 3 // Info details view
		case "p":
			// Prune system
			return m, m.pruneSystem()
		case "r":
			// Refresh information
			return m, tea.Batch(
				m.fetchSystemInfo(),
				m.fetchDiskUsage(),
			)
		}

	case SystemInfoMsg:
		m.loading = false
		if msg.Error != nil {
			m.error = msg.Error
			return m, nil
		}
		m.info = msg.Info
		m.version = msg.Version

	case SystemDiskUsageMsg:
		m.loading = false
		if msg.Error != nil {
			m.error = msg.Error
			return m, nil
		}
		m.diskUsage = msg.DiskUsage
	}

	return m, nil
}

// View renders the model
func (m *SystemModel) View() string {
	if m.loading {
		return StyleMainLayout.Render("Loading system information...")
	}

	if m.error != nil {
		errorBox := StyleInfoBox.
			BorderForeground(ColorError).
			Render(StyleError.Render("Error loading system info: " + m.error.Error()))
		
		helpText := "\nPress 'esc' to go back to the main menu."
		
		return StyleMainLayout.Render(
			lipgloss.JoinVertical(lipgloss.Center,
				StyleTitle.Render("System Management Error"),
				errorBox,
				helpText,
			),
		)
	}

	title := StyleTitle.Render("System Management")

	// Build menu
	menu := []string{
		"1. System Overview",
		"2. Disk Usage",
		"3. Version Details",
		"4. Info Details",
	}
	menuText := strings.Join(menu, " • ")
	menuBox := StyleInfoBox.Render(menuText)

	var content string
	switch m.subView {
	case 0: // Main system view
		content = m.renderMainView()
	case 1: // Disk usage view
		content = m.renderDiskUsageView()
	case 2: // Version details view
		content = m.renderVersionView()
	case 3: // Info details view
		content = m.renderInfoView()
	}

	// Create help text
	helpText := StyleHelp.Render(
		"1-4: Switch views • r: Refresh • p: Prune system • esc: Back",
	)

	// Join everything together
	fullContent := lipgloss.JoinVertical(lipgloss.Left,
		title,
		"",
		menuBox,
		"",
		content,
		"",
		helpText,
	)

	return StyleMainLayout.Render(fullContent)
}

// renderMainView renders the main system overview
func (m *SystemModel) renderMainView() string {
	// Format system overview info
	infoContent := fmt.Sprintf(
		"Docker Version: %s\nAPI Version: %s\nOS/Arch: %s/%s\nKernel: %s\n"+
			"CPU Cores: %d\nTotal Memory: %s\nStorage Driver: %s",
		m.version.Version,
		m.version.APIVersion,
		m.info.OperatingSystem,
		m.info.Architecture,
		m.info.KernelVersion,
		m.info.NCPU,
		formatter.FormatSize(float64(m.info.MemTotal)),
		m.info.Driver,
	)
	
	usageContent := fmt.Sprintf(
		"Images: %d\nContainers: %d\nVolumes: %d\n"+
			"Total Size: %s\nImages Size: %s",
		len(m.diskUsage.Images),
		len(m.diskUsage.Containers),
		len(m.diskUsage.Volumes),
		formatter.FormatSize(float64(m.diskUsage.LayersSize)),
		formatter.FormatSize(float64(m.diskUsage.LayersSize)), // Approximate
	)
	
	infoBox := StyleInfoBox.Width(40).Render(infoContent)
	usageBox := StyleInfoBox.Width(40).Render(usageContent)
	
	return lipgloss.JoinHorizontal(lipgloss.Top, infoBox, usageBox)
}

// renderDiskUsageView renders the disk usage details
func (m *SystemModel) renderDiskUsageView() string {
	// Format disk usage info for images
	var imageRows []string
	imageRows = append(imageRows, lipgloss.JoinHorizontal(lipgloss.Left,
		StyleTableHeader.Width(40).Render("IMAGE"),
		StyleTableHeader.Width(20).Render("SIZE"),
		StyleTableHeader.Width(20).Render("CREATED"),
	))
	
	for _, img := range m.diskUsage.Images {
		// Truncate image name if necessary
		imageName := "<none>:<none>"
		if len(img.RepoTags) > 0 {
			imageName = img.RepoTags[0]
			if len(imageName) > 39 {
				imageName = imageName[:36] + "..."
			}
		}
		
		// Format creation time
		created := time.Unix(img.Created, 0)
		createdStr := formatter.FormatTime(created)
		
		imageRows = append(imageRows, lipgloss.JoinHorizontal(lipgloss.Left,
			StyleTableRow.Width(40).Render(imageName),
			StyleTableRow.Width(20).Render(formatter.FormatSize(float64(img.Size))),
			StyleTableRow.Width(20).Render(createdStr),
		))
	}
	
	return strings.Join(imageRows, "\n")
}

// renderVersionView renders the version details
func (m *SystemModel) renderVersionView() string {
	// Format version info
	versionContent := fmt.Sprintf(
		"Version: %s\nAPI Version: %s\nMin API Version: %s\n"+
			"Git Commit: %s\nGo Version: %s\n"+
			"OS/Arch: %s/%s\nExperimental: %v\nBuild Time: %s",
		m.version.Version,
		m.version.APIVersion,
		m.version.MinAPIVersion,
		m.version.GitCommit,
		m.version.GoVersion,
		m.version.Os,
		m.version.Arch,
		m.version.Experimental,
		m.version.BuildTime,
	)
	
	return StyleInfoBox.Render(versionContent)
}

// renderInfoView renders the detailed system info
func (m *SystemModel) renderInfoView() string {
	// Format system info
	infoContent := fmt.Sprintf(
		"ID: %s\nContainers: %d\nRunning: %d\nPaused: %d\nStopped: %d\n"+
			"Images: %d\nDriver: %s\n"+
			"System Time: %s\nKernel Version: %s\n"+
			"Operating System: %s\nArchitecture: %s\nCPU Cores: %d\n"+
			"Memory: %s\nCgroup Driver: %s\nCgroup Version: %s",
		m.info.ID,
		m.info.Containers,
		m.info.ContainersRunning,
		m.info.ContainersPaused,
		m.info.ContainersStopped,
		m.info.Images,
		m.info.Driver,
		time.Now().Format(time.RFC3339),
		m.info.KernelVersion,
		m.info.OperatingSystem,
		m.info.Architecture,
		m.info.NCPU,
		formatter.FormatSize(float64(m.info.MemTotal)),
		m.info.CgroupDriver,
		m.info.CgroupVersion,
	)
	
	return StyleInfoBox.Render(infoContent)
}