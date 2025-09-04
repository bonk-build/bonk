// Copyright © 2025 Colden Cullen
// SPDX-License-Identifier: MIT

package main

import (
	"log/slog"
	"os"
	"path"

	"github.com/spf13/cobra"

	"go.bonk.build/pkg/driver"
	"go.bonk.build/pkg/task"
	"go.bonk.build/plugins/k8s/holos"
)

var (
	platform    string
	concurrency uint
)

// rootCmd represents the base command when called without any subcommands.
var rootCmd = &cobra.Command{
	Use:   "holos",
	Short: "Holos-compatible bonk driver",

	Run: func(cmd *cobra.Command, args []string) {
		sessionDir, _ := os.Getwd()

		if len(args) > 0 {
			sessionDir = path.Join(sessionDir, args[0])
		}

		err := driver.Run(cmd.Context(),
			driver.WithConcurrency(concurrency),
			driver.WithGenericExecutor(holos.Plugin.Name(), holos.Plugin),
			driver.WithPlugins(
				"go.bonk.build/plugins/k8s/resources",
				"go.bonk.build/plugins/k8s/kustomize",
			),
			driver.WithLocalSession(sessionDir,
				driver.WithTask(
					"platform",
					"holos.RenderPlatform",
					map[string]any{
						"platform": platform,
					},
					task.WithInputs(path.Join(platform, "*.cue")),
				),
			),
		)
		cobra.CheckErr(err)
	},
}

func init() {
	rootCmd.PersistentFlags().
		StringVarP(&platform, "platform", "p", "platform", "The default platform directory to use")
	rootCmd.PersistentFlags().
		UintVarP(&concurrency, "concurrency", "j", 100, "The number of goroutines to run") //nolint:mnd
}

func main() {
	err := rootCmd.Execute()
	if err != nil {
		slog.Error(err.Error())
		os.Exit(1)
	}
}
