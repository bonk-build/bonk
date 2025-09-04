// Copyright © 2025 Colden Cullen
// SPDX-License-Identifier: MIT

package main

import (
	"log/slog"
	"os"
	"path"

	"github.com/pterm/pterm"
	"github.com/spf13/cobra"

	"go.bonk.build/pkg/executor/plugin"
	"go.bonk.build/pkg/scheduler"
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

		pum := plugin.NewPluginClientManager()

		err := pum.StartPlugins(cmd.Context(),
			"go.bonk.build/plugins/k8s/resources",
			"go.bonk.build/plugins/k8s/kustomize",
		)
		cobra.CheckErr(err)

		err = pum.RegisterExecutor(holos.Plugin.Name(), holos.Plugin)
		cobra.CheckErr(err)

		session := task.NewLocalSession(task.NewSessionId(), sessionDir)
		err = pum.OpenSession(cmd.Context(), session)
		cobra.CheckErr(err)

		sched := scheduler.NewScheduler(pum, concurrency)

		err = sched.AddTask(
			task.New(
				session,
				"holos.RenderPlatform",
				"platform",
				map[string]any{
					"platform": platform,
				},
				path.Join(platform, "*.cue"),
			).Box(),
		)
		cobra.CheckErr(err)

		sched.Run()
		pum.CloseSession(cmd.Context(), session.ID())
		pum.Shutdown()
		pum.Shutdown()
	},
}

func init() {
	rootCmd.PersistentFlags().
		StringVarP(&platform, "platform", "p", "platform", "The default platform directory to use")
	rootCmd.PersistentFlags().
		UintVarP(&concurrency, "concurrency", "j", 100, "The number of goroutines to run") //nolint:mnd
}

func main() {
	slog.SetDefault(
		slog.New(
			pterm.NewSlogHandler(
				pterm.DefaultLogger.
					WithWriter(rootCmd.OutOrStdout()).
					WithLevel(pterm.LogLevelDebug).
					WithTime(false),
			),
		),
	)

	err := rootCmd.Execute()
	if err != nil {
		slog.Error(err.Error())
		os.Exit(1)
	}
}
