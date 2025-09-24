// Copyright © 2025 Colden Cullen
// SPDX-License-Identifier: MIT

package bubbletea

import (
	"github.com/charmbracelet/lipgloss/v2"

	"go.bonk.build/pkg/executor/observer"
)

type StatusStyles map[observer.TaskStatus]lipgloss.Style

var (
	StatusStyleClear = StatusStyles{
		observer.StatusNone:    lipgloss.NewStyle().SetString("  ").Faint(true),
		observer.StatusRunning: lipgloss.NewStyle().SetString("🔘 "),
		observer.StatusSuccess: lipgloss.NewStyle().SetString("✔️ ").Foreground(lipgloss.Green),
		observer.StatusError:   lipgloss.NewStyle().SetString("❌ ").Foreground(lipgloss.Red),
	}
	StatusStyleCircle = StatusStyles{
		observer.StatusNone:    lipgloss.NewStyle().SetString("  ").Faint(true),
		observer.StatusRunning: lipgloss.NewStyle().SetString("🔵 "),
		observer.StatusSuccess: lipgloss.NewStyle().SetString("🟢 ").Foreground(lipgloss.Green),
		observer.StatusError:   lipgloss.NewStyle().SetString("🔴 ").Foreground(lipgloss.Red),
	}
)
