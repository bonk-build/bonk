// Copyright © 2025 Colden Cullen
// SPDX-License-Identifier: MIT

package main

import (
	"log/slog"
	"os"
	"path"

	"github.com/spf13/cobra"

	"go.bonk.build/pkg/driver"
	"go.bonk.build/pkg/observer/bubbletea"
	"go.bonk.build/pkg/task"
	"go.bonk.build/plugins/k8s/holos"
)

var (
	platform    string
	concurrency int
)

// rootCmd represents the base command when called without any subcommands.
var rootCmd = &cobra.Command{
	Use:   "holos",
	Short: "Holos-compatible bonk driver",

	RunE: func(cmd *cobra.Command, args []string) error {
		sessionDir, _ := os.Getwd()

		if len(args) > 0 {
			sessionDir = path.Join(sessionDir, args[0])
		}

		bubble := bubbletea.New(cmd.Context(), true)

		var result task.Result
		err := driver.Run(cmd.Context(), &result, driver.MakeDefaultOptions().
			WithConcurrency(concurrency).
			WithObservers(bubble.OnTaskStatusMsg).
			WithExecutor(holos.Plugin.Name(), holos.Plugin).
			WithPlugins(
				"go.bonk.build/plugins/k8s/resources",
				"go.bonk.build/plugins/k8s/kustomize",
			).
			WithLocalSession(sessionDir,
				task.New(
					task.NewID("platform"),
					"holos.RenderPlatform",
					map[string]any{
						"platform": platform,
					},
					task.WithInputs(task.SourceFile(path.Join(platform, "*.cue"))),
				),
			))
		if err != nil {
			return err
		}

		bubble.Quit()

		return nil
	},
}

func init() {
	rootCmd.PersistentFlags().
		StringVarP(&platform, "platform", "p", "platform", "The default platform directory to use")
	rootCmd.PersistentFlags().
		IntVarP(&concurrency, "concurrency", "j", 100, "The number of goroutines to run") //nolint:mnd
}

func main() {
	err := rootCmd.Execute()
	if err != nil {
		slog.Error(err.Error())
		os.Exit(1)
	}
}
