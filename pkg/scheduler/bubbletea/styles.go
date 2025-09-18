// Copyright © 2025 Colden Cullen
// SPDX-License-Identifier: MIT

package bubbletea

import "github.com/charmbracelet/lipgloss/v2"

type StatusStyles map[TaskStatus]lipgloss.Style

var (
	StatusStyleClear = StatusStyles{
		StatusNone:      lipgloss.NewStyle().SetString("  ").Faint(true),
		StatusScheduled: lipgloss.NewStyle().SetString("🔘 "),
		StatusSuccess:   lipgloss.NewStyle().SetString("✔️ ").Foreground(lipgloss.Green),
		StatusFail:      lipgloss.NewStyle().SetString("❌ ").Foreground(lipgloss.Red),
	}
	StatusStyleCircle = StatusStyles{
		StatusNone:      lipgloss.NewStyle().SetString("  ").Faint(true),
		StatusScheduled: lipgloss.NewStyle().SetString("🔵 "),
		StatusSuccess:   lipgloss.NewStyle().SetString("🟢 ").Foreground(lipgloss.Green),
		StatusFail:      lipgloss.NewStyle().SetString("🔴 ").Foreground(lipgloss.Red),
	}
)
