package main

import (
	"fmt"
	"os"

	"github.com/Gostatsog/dockerNav/internal/ui"
	tea "github.com/charmbracelet/bubbletea"
)

func main() {
	// Initialize the Bubble Tea program with our main UI model
	p := tea.NewProgram(ui.NewMainModel(), tea.WithAltScreen())
	
	// Run the program
	if _, err := p.Run(); err != nil {
		fmt.Printf("Error running app: %v\n", err)
		os.Exit(1)
	}
}