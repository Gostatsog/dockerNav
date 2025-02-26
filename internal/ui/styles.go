package ui

import (
	"github.com/charmbracelet/lipgloss"
)

// Color scheme
var (
	ColorPrimary     = lipgloss.Color("#7D56F4")  // Main accent color
	ColorSecondary   = lipgloss.Color("#FC8EAC")  // Second accent color
	ColorBackground  = lipgloss.Color("#282828")  // Dark background
	ColorText        = lipgloss.Color("#FFFFFF")  // Default text color
	ColorSubtle      = lipgloss.Color("#888888")  // Subtle/secondary text
	ColorSuccess     = lipgloss.Color("#73F59F")  // Success messages
	ColorWarning     = lipgloss.Color("#F5A623")  // Warning messages
	ColorError       = lipgloss.Color("#F5426C")  // Error messages
	ColorHighlight   = lipgloss.Color("#00B7C3")  // Highlight color
)

// Common styles
var (
	// Title style for section headers
	StyleTitle = lipgloss.NewStyle().
		Bold(true).
		Foreground(ColorText).
		Background(ColorPrimary).
		Padding(0, 1).
		MarginBottom(1)

	// Info box style for displaying data
	StyleInfoBox = lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(ColorPrimary).
		Padding(1)

	// Menu style for navigation menus
	StyleMenu = lipgloss.NewStyle().
		Padding(1)

	// Footer style for help text
	StyleFooter = lipgloss.NewStyle().
		Foreground(ColorSubtle)

	// Main layout container
	StyleMainLayout = lipgloss.NewStyle().
		Padding(1, 2)

	// Error message style
	StyleError = lipgloss.NewStyle().
		Foreground(ColorError).
		Bold(true)

	// Success message style
	StyleSuccess = lipgloss.NewStyle().
		Foreground(ColorSuccess).
		Bold(true)
		
	// Warning message style
	StyleWarning = lipgloss.NewStyle().
		Foreground(ColorWarning).
		Bold(true)
		
	// Selected item style
	StyleSelected = lipgloss.NewStyle().
		Foreground(ColorText).
		Background(ColorPrimary).
		Bold(true)
		
	// Button style
	StyleButton = lipgloss.NewStyle().
		Foreground(ColorText).
		Background(ColorPrimary).
		Padding(0, 3).
		Margin(0, 1).
		Bold(true)
		
	// Focused button style
	StyleButtonFocused = StyleButton.Copy().
		Background(ColorHighlight)
		
	// Table header style
	StyleTableHeader = lipgloss.NewStyle().
		BorderStyle(lipgloss.NormalBorder()).
		BorderForeground(ColorPrimary).
		BorderBottom(true).
		Bold(true)
		
	// Table row style
	StyleTableRow = lipgloss.NewStyle().
		Padding(0, 1)
)