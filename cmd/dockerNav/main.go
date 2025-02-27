package main

import (
	"fmt"
	"os"

	"golang.org/x/term"

	"github.com/Gostatsog/dockerNav/internal/ui"
	tea "github.com/charmbracelet/bubbletea"
)

func main() {
	// Query the actual terminal size.
	width, height, err := term.GetSize(int(os.Stdout.Fd()))
	if err != nil {
		// Fallback to a reasonable default.
		width, height = 120, 40
	}

	// Initialize the main model with actual dimensions.
	m := ui.NewMainModel(width, height)
	
	// Initialize the Bubble Tea program.
	p := tea.NewProgram(m, tea.WithAltScreen())

	// Run the program.
	if _, err := p.Run(); err != nil {
		fmt.Printf("Error running app: %v\n", err)
		os.Exit(1)
	}
}