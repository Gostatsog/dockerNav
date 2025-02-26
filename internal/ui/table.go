package ui

import (
	"os"

	"github.com/olekukonko/tablewriter"
)

// TableOptions represents the options for a table
type TableOptions struct {
	Headers      []string
	Rows         [][]string
	HeaderColors []tablewriter.Colors
	RowColors    [][]tablewriter.Colors
	Border       bool
	RowLine      bool
}

// DefaultHeaderColors returns default header colors (cyan bold)
func DefaultHeaderColors(headerCount int) []tablewriter.Colors {
	colors := make([]tablewriter.Colors, headerCount)
	for i := range colors {
		colors[i] = tablewriter.Colors{tablewriter.Bold, tablewriter.FgCyanColor}
	}
	return colors
}

// RenderTable renders a table with the given options
func RenderTable(options TableOptions) {
	table := tablewriter.NewWriter(os.Stdout)
	table.SetHeader(options.Headers)
	table.SetBorder(options.Border)
	table.SetRowLine(options.RowLine)

	// Apply header colors if provided
	if options.HeaderColors != nil && len(options.HeaderColors) > 0 {
		table.SetHeaderColor(options.HeaderColors...)
	}

	// Apply row colors if provided
	if options.RowColors != nil && len(options.RowColors) > 0 {
		for i, row := range options.Rows {
			if i < len(options.RowColors) {
				table.Rich(row, options.RowColors[i])
			} else {
				table.Append(row)
			}
		}
	} else {
		table.AppendBulk(options.Rows)
	}

	table.Render()
}

// CreateTable creates a new table with common settings
func CreateTable(headers []string) *tablewriter.Table {
	table := tablewriter.NewWriter(os.Stdout)
	table.SetHeader(headers)
	table.SetBorder(false)
	table.SetHeaderColor(DefaultHeaderColors(len(headers))...)
	return table
}