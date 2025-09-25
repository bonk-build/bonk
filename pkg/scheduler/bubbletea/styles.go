// Copyright © 2025 Colden Cullen
// SPDX-License-Identifier: MIT

package bubbletea

import (
	"github.com/charmbracelet/lipgloss/v2"

	"go.bonk.build/pkg/executor/observable"
)

type StatusStyles map[observable.TaskStatus]lipgloss.Style

var (
	StatusStyleClear = StatusStyles{
		observable.StatusNone:    lipgloss.NewStyle().SetString("  ").Faint(true),
		observable.StatusRunning: lipgloss.NewStyle().SetString("🔘 "),
		observable.StatusSuccess: lipgloss.NewStyle().SetString("✔️ ").Foreground(lipgloss.Green),
		observable.StatusError:   lipgloss.NewStyle().SetString("❌ ").Foreground(lipgloss.Red),
	}
	StatusStyleCircle = StatusStyles{
		observable.StatusNone:    lipgloss.NewStyle().SetString("  ").Faint(true),
		observable.StatusRunning: lipgloss.NewStyle().SetString("🔵 "),
		observable.StatusSuccess: lipgloss.NewStyle().SetString("🟢 ").Foreground(lipgloss.Green),
		observable.StatusError:   lipgloss.NewStyle().SetString("🔴 ").Foreground(lipgloss.Red),
	}
)
