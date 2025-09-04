// Copyright © 2025 Colden Cullen
// SPDX-License-Identifier: MIT

package main

import (
	"log/slog"
	"os"
	"path"

	"cuelang.org/go/cue/cuecontext"

	"github.com/pterm/pterm"
	"github.com/spf13/cobra"

	"go.bonk.build/pkg/executor"
	"go.bonk.build/pkg/plugin"
	"go.bonk.build/pkg/scheduler"
	"go.bonk.build/pkg/task"
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

		cuectx := cuecontext.New()

		bem := executor.NewExecutorManager("")
		pum := plugin.NewPluginManager(cuectx, &bem)

		err := pum.StartPlugins(cmd.Context(),
			"go.bonk.build/plugins/k8s/holos",
			"go.bonk.build/plugins/k8s/resources",
			"go.bonk.build/plugins/k8s/kustomize",
		)
		cobra.CheckErr(err)

		session := task.NewLocalSession(sessionDir)
		err = bem.OpenSession(cmd.Context(), session)
		cobra.CheckErr(err)

		sched := scheduler.NewScheduler(&bem, concurrency)

		err = sched.AddTask(
			task.New(
				session,
				"holos.RenderPlatform",
				"platform",
				cuectx.CompileString(`platform: "`+platform+`"`),
				path.Join(platform, "*.cue"),
			),
		)
		cobra.CheckErr(err)

		sched.Run()
		bem.CloseSession(cmd.Context(), session.ID())
		pum.Shutdown()
		bem.Shutdown()
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
