package main

import (
	"fmt"
	"os"

	"github.com/Gostatsog/dockerNav/internal/app"
	"github.com/spf13/cobra"
)

func main() {
	// Root command
	var rootCmd = &cobra.Command{
		Use:   "dockergo",
		Short: "DockerGo - Interactive Docker CLI",
		Long:  `DockerGo is an interactive CLI tool for Docker that simplifies container, image, network, and volume management.`,
		Run: func(cmd *cobra.Command, args []string) {
			// Initialize and start the application
			application, err := app.NewApp()
			if err != nil {
				fmt.Printf("Error initializing application: %v\n", err)
				os.Exit(1)
			}
			
			// Start the main menu
			application.StartMainMenu()
		},
	}

	if err := rootCmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}