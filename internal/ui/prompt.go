package ui

import (
	"bufio"
	"fmt"
	"os"
	"strings"

	"github.com/manifoldco/promptui"
)

// SelectOptions represents options for a selection prompt
type SelectOptions struct {
	Label    string
	Items    []string
	Size     int
	HelpText string
}

// DefaultSelectTemplates returns the default select templates
func DefaultSelectTemplates() *promptui.SelectTemplates {
	return &promptui.SelectTemplates{
		Label:    "{{ . }}",
		Active:   "🔹 {{ . | cyan }}",
		Inactive: "  {{ . | white }}",
		Selected: "🔸 {{ . | green }}",
		Help:     "{{ \"Use arrow keys to navigate, enter to select\" | faint }}",
	}
}

// Select displays a selection prompt and returns the selected item
func Select(options SelectOptions) (int, string, error) {
	size := 10 // Default size
	if options.Size > 0 {
		size = options.Size
	}

	prompt := promptui.Select{
		Label:     options.Label,
		Items:     options.Items,
		Templates: DefaultSelectTemplates(),
		Size:      size,
	}

	return prompt.Run()
}

// PromptOptions represents options for a text input prompt
type PromptOptions struct {
	Label     string
	Default   string
	Validate  func(string) error
	IsConfirm bool
}

// Prompt displays a text input prompt and returns the entered text
func Prompt(options PromptOptions) (string, error) {
	prompt := promptui.Prompt{
		Label:    options.Label,
		Default:  options.Default,
		Validate: options.Validate,
	}

	return prompt.Run()
}

// Confirm displays a yes/no confirmation prompt
func Confirm(message string) (bool, error) {
	prompt := promptui.Select{
		Label:     message,
		Items:     []string{"Yes", "No"},
		Templates: DefaultSelectTemplates(),
	}

	_, result, err := prompt.Run()
	if err != nil {
		return false, err
	}

	return result == "Yes", nil
}

// WaitForEnter waits for the user to press Enter
func WaitForEnter() {
	fmt.Println("\nPress Enter to continue...")
	bufio.NewReader(os.Stdin).ReadBytes('\n')
}

// ClearScreen clears the terminal screen
func ClearScreen() {
	fmt.Print("\033[H\033[2J")
}

// ReadInput reads a line of text from stdin
func ReadInput(prompt string) string {
	fmt.Print(prompt)
	reader := bufio.NewReader(os.Stdin)
	input, _ := reader.ReadString('\n')
	return strings.TrimSpace(input)
}