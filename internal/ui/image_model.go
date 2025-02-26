package ui

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/Gostatsog/dockerNav/internal/client"
	"github.com/Gostatsog/dockerNav/pkg/formatter"
	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/list"
	"github.com/charmbracelet/bubbles/spinner"
	"github.com/charmbracelet/bubbles/textinput"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/docker/docker/api/types/image"
)

// ImageListMsg carries image data after fetching
type ImageListMsg struct {
	Images []image.Summary
	Error  error
}

// ImageActionMsg carries results of image actions
type ImageActionMsg struct {
	Action  string
	ImageID string
	Error   error
}

// ImagePullMsg carries results of pulling an image
type ImagePullMsg struct {
	Success bool
	Error   error
}

// ImageItem represents an image in the list
type ImageItem struct {
	image image.Summary
	title string
	desc  string
}

// FilterValue implements list.Item interface
func (i ImageItem) FilterValue() string { return i.title }

// Title returns the title for the list item
func (i ImageItem) Title() string { return i.title }

// Description returns the description for the list item
func (i ImageItem) Description() string { return i.desc }

// ImageKeyMap defines keybindings for image operations
type ImageKeyMap struct {
	Refresh key.Binding
	Pull    key.Binding
	Remove  key.Binding
	Back    key.Binding
}

// DefaultImageKeyMap returns default image keybindings
func DefaultImageKeyMap() ImageKeyMap {
	return ImageKeyMap{
		Refresh: key.NewBinding(
			key.WithKeys("r"),
			key.WithHelp("r", "refresh"),
		),
		Pull: key.NewBinding(
			key.WithKeys("p"),
			key.WithHelp("p", "pull"),
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

// ImageModel manages image view state
type ImageModel struct {
	docker    *client.DockerClient
	imageList list.Model
	keyMap    ImageKeyMap
	state     string // "list", "pull", "confirm"
	width     int
	height    int
	spin      spinner.Model
	viewport  viewport.Model
	textInput textinput.Model
	selectedImage *image.Summary
	confirmMsg   string
	confirmAction string
	loading    bool
	error      error
}

// NewImageModel creates a new image model
func NewImageModel(docker *client.DockerClient) *ImageModel {
	keyMap := DefaultImageKeyMap()
	
	// Set up image list
	delegate := list.NewDefaultDelegate()
	imageList := list.New([]list.Item{}, delegate, 0, 0)
	imageList.Title = "Images"
	imageList.Styles.Title = StyleTitle
	imageList.AdditionalFullHelpKeys = func() []key.Binding {
		return []key.Binding{
			keyMap.Refresh,
			keyMap.Pull,
			keyMap.Remove,
			keyMap.Back,
		}
	}

	// Set up spinner for loading states
	s := spinner.New()
	s.Spinner = spinner.Dot
	s.Style = lipgloss.NewStyle().Foreground(ColorPrimary)

	// Setup text input for pulling images
	ti := textinput.New()
	ti.Placeholder = "image:tag (e.g., nginx:latest)"
	ti.Width = 40
	ti.Focus()

	// Set up viewport for scrollable content
	vp := viewport.New(0, 0)
	vp.Style = lipgloss.NewStyle().
		BorderStyle(lipgloss.RoundedBorder()).
		BorderForeground(ColorPrimary)

	return &ImageModel{
		docker:    docker,
		imageList: imageList,
		keyMap:    keyMap,
		state:     "list",
		spin:      s,
		viewport:  vp,
		textInput: ti,
		loading:   true,
	}
}

// Init initializes the model
func (m *ImageModel) Init() tea.Cmd {
	return tea.Batch(m.fetchImages(), m.spin.Tick)
}

// fetchImages returns a command that fetches image data
func (m *ImageModel) fetchImages() tea.Cmd {
	return func() tea.Msg {
		ctx := context.Background()
		images, err := m.docker.Client.ImageList(ctx, image.ListOptions{})
		return ImageListMsg{
			Images: images,
			Error:  err,
		}
	}
}

// pullImage returns a command that pulls an image
func (m *ImageModel) pullImage(imageName string) tea.Cmd {
	return func() tea.Msg {
		ctx := context.Background()
		reader, err := m.docker.Client.ImagePull(ctx, imageName, image.PullOptions{})
		if err != nil {
			return ImagePullMsg{Success: false, Error: err}
		}
		defer reader.Close()

		// We need to read the response to complete the pull
		buf := new(strings.Builder)
		_, err = buf.ReadFrom(reader)
		if err != nil {
			return ImagePullMsg{Success: false, Error: err}
		}

		return ImagePullMsg{Success: true, Error: nil}
	}
}

// performImageAction returns a command that performs an action on an image
func (m *ImageModel) performImageAction(action string, imageID string) tea.Cmd {
	return func() tea.Msg {
		ctx := context.Background()
		var err error
		
		switch action {
		case "remove":
			_, err = m.docker.Client.ImageRemove(ctx, imageID, image.RemoveOptions{})
		}
		
		return ImageActionMsg{
			Action:  action,
			ImageID: imageID,
			Error:   err,
		}
	}
}

// Update handles messages and updates the model
func (m *ImageModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmds []tea.Cmd

	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch m.state {
		case "list":
			switch {
			case key.Matches(msg, m.keyMap.Refresh):
				m.loading = true
				return m, tea.Batch(m.fetchImages(), m.spin.Tick)
				
			case key.Matches(msg, m.keyMap.Pull):
				m.state = "pull"
				m.textInput.Focus()
				return m, nil
				
			case key.Matches(msg, m.keyMap.Remove):
				if item, ok := m.imageList.SelectedItem().(ImageItem); ok {
					m.selectedImage = &item.image
					m.confirmMsg = fmt.Sprintf("Are you sure you want to remove image %s?", item.title)
					m.confirmAction = "remove"
					m.state = "confirm"
					return m, nil
				}
			}
			
		case "pull":
			switch msg.String() {
			case "enter":
				imageName := m.textInput.Value()
				if imageName == "" {
					m.state = "list"
					return m, nil
				}
				m.loading = true
				m.state = "list"
				return m, tea.Batch(m.pullImage(imageName), m.spin.Tick)
				
			case "esc":
				m.state = "list"
				m.textInput.Reset()
				return m, nil
			}
			
			var cmd tea.Cmd
			m.textInput, cmd = m.textInput.Update(msg)
			return m, cmd
			
		case "confirm":
			switch msg.String() {
			case "y", "Y":
				if m.selectedImage != nil {
					return m, m.performImageAction(m.confirmAction, m.selectedImage.ID)
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
		m.imageList.SetSize(msg.Width-4, msg.Height-headerHeight-footerHeight)
		
		// Update viewport dimensions
		m.viewport.Width = msg.Width - 4
		m.viewport.Height = msg.Height - headerHeight - footerHeight

	case spinner.TickMsg:
		var cmd tea.Cmd
		m.spin, cmd = m.spin.Update(msg)
		cmds = append(cmds, cmd)

	case ImageListMsg:
		m.loading = false
		if msg.Error != nil {
			m.error = msg.Error
			return m, nil
		}

		items := make([]list.Item, 0, len(msg.Images))
		for _, img := range msg.Images {
			// Handle images with no repository/tag
			name := "<none>:<none>"
			if len(img.RepoTags) > 0 && img.RepoTags[0] != "<none>:<none>" {
				name = img.RepoTags[0]
			}
			
			// Format created time
			createdTime := time.Unix(img.Created, 0)
			created := formatter.FormatTime(createdTime)
			
			// Format size
			size := formatter.FormatSize(float64(img.Size))
			
			desc := fmt.Sprintf("ID: %s • Created: %s • Size: %s",
				img.ID[7:19],
				created,
				size,
			)
			
			items = append(items, ImageItem{
				image: img,
				title: name,
				desc:  desc,
			})
		}
		
		var cmd tea.Cmd
		m.imageList.SetItems(items)
		return m, cmd
		
	case ImagePullMsg:
		m.loading = false
		if msg.Error != nil {
			m.error = msg.Error
			return m, nil
		}
		
		// Refresh image list after successful pull
		return m, m.fetchImages()
		
	case ImageActionMsg:
		m.loading = false
		if msg.Error != nil {
			m.error = msg.Error
			return m, nil
		}
		
		// Refresh image list after successful action
		m.state = "list"
		return m, m.fetchImages()
	}

	// Update list in list state
	if m.state == "list" {
		var cmd tea.Cmd
		m.imageList, cmd = m.imageList.Update(msg)
		cmds = append(cmds, cmd)
	}

	return m, tea.Batch(cmds...)
}

// View renders the current view
func (m *ImageModel) View() string {
	if m.loading {
		return StyleMainLayout.Render(
			lipgloss.JoinVertical(lipgloss.Center,
				StyleTitle.Render("Images"),
				fmt.Sprintf("%s Loading images...", m.spin.View()),
			),
		)
	}

	if m.error != nil {
		errorBox := StyleInfoBox.Copy().
			BorderForeground(ColorError).
			Render(StyleError.Render(fmt.Sprintf("Error: %v", m.error)))
		
		help := "Press r to retry, esc to go back"
		return StyleMainLayout.Render(
			lipgloss.JoinVertical(lipgloss.Left,
				StyleTitle.Render("Image Management"),
				errorBox, 
				help,
			),
		)
	}

	var content string
	switch m.state {
	case "list":
		content = lipgloss.JoinVertical(lipgloss.Left,
			StyleTitle.Render("Image Management"),
			"",
			m.imageList.View(),
		)
		
	case "pull":
		inputBox := StyleInfoBox.Render(
			lipgloss.JoinVertical(lipgloss.Left,
				"Enter image to pull:",
				m.textInput.View(),
				"",
				"Press Enter to pull or Esc to cancel",
			),
		)
		
		content = lipgloss.JoinVertical(lipgloss.Left,
			StyleTitle.Render("Pull Image"),
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

	return StyleMainLayout.Render(content)
}