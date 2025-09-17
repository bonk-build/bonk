// Copyright © 2025 Colden Cullen
// SPDX-License-Identifier: MIT

package bubbletea

import "github.com/charmbracelet/lipgloss/v2"

type StatusStyle struct {
	lipgloss.Style

	Emoji string
}

type StatusStyles map[TaskStatus]StatusStyle

var (
	StatusStyleClear = StatusStyles{
		StatusScheduled: StatusStyle{
			Style: lipgloss.NewStyle(),
			Emoji: "🔘",
		},
		StatusSuccess: StatusStyle{
			Style: lipgloss.NewStyle().Foreground(lipgloss.Green),
			Emoji: "✔️",
		},
		StatusFail: StatusStyle{
			Style: lipgloss.NewStyle().Foreground(lipgloss.Red),
			Emoji: "❌",
		},
	}
	StatusStyleCircle = StatusStyles{
		StatusScheduled: StatusStyle{
			Style: lipgloss.NewStyle(),
			Emoji: "🔵",
		},
		StatusSuccess: StatusStyle{
			Style: lipgloss.NewStyle().Foreground(lipgloss.Green),
			Emoji: "🟢",
		},
		StatusFail: StatusStyle{
			Style: lipgloss.NewStyle().Foreground(lipgloss.Red),
			Emoji: "🔴",
		},
	}
)
