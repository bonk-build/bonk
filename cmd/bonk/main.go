// Copyright © 2025 Colden Cullen
// SPDX-License-Identifier: MIT

package main

import (
	"log/slog"
	"os"
	"path"

	"go.uber.org/multierr"

	"github.com/pterm/pterm"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"

	slogmulti "github.com/samber/slog-multi"
	slogctx "github.com/veqryn/slog-context"

	"go.bonk.build/pkg/executor/plugin"
	"go.bonk.build/pkg/scheduler"
	"go.bonk.build/pkg/task"
)

var (
	cfgFile     string
	concurrency uint
)

// rootCmd represents the base command when called without any subcommands.
var rootCmd = &cobra.Command{
	Use:   "bonk",
	Short: "A cue-based configuration build system.",

	Run: func(cmd *cobra.Command, args []string) {
		pum := plugin.NewPluginClientManager()
		defer pum.Shutdown()

		sched := scheduler.NewScheduler(pum, concurrency)

		err := pum.StartPlugins(cmd.Context(),
			"go.bonk.build/plugins/test",
			"go.bonk.build/plugins/k8s/resources",
			"go.bonk.build/plugins/k8s/kustomize",
		)
		cobra.CheckErr(err)

		cwd, _ := os.Getwd()
		session := task.NewLocalSession(task.NewSessionId(), path.Join(cwd, "testdata"))

		err = pum.OpenSession(cmd.Context(), session)
		defer pum.CloseSession(cmd.Context(), session.ID())
		cobra.CheckErr(err)

		cobra.CheckErr(multierr.Combine(
			sched.AddTask(
				task.New(
					session,
					"test.Test",
					"Test.Test",
					map[string]any{
						"value": 3,
					},
				).Box(),
			),
			sched.AddTask(
				task.New(
					session,
					"resources.Resources",
					"Test.Resources",
					map[string]any{
						"resources": []map[string]any{
							{
								"apiVersion": "v1",
								"kind":       "Namespace",
								"metadata": map[string]any{
									"name": "Testing",
								},
							},
						},
					},
				).Box(),
			),
			sched.AddTask(
				task.New(
					session,
					"kustomize.Kustomize",
					"Test.Kustomize",
					map[string]any{},
					".bonk/Test.Resources/resources.yaml",
				).Box(),
				"Test.Resources",
			),
		))

		sched.Run()
	},
}

func init() {
	rootCmd.PersistentFlags().
		StringVarP(&cfgFile, "config", "c", "", "config file (default is .bonk.yaml)")
	rootCmd.PersistentFlags().
		UintVarP(&concurrency, "concurrency", "j", 100, "The number of goroutines to run")

	if cfgFile != "" {
		// Use config file from the flag.
		viper.SetConfigFile(cfgFile)
	} else {
		// Search config in current directory with name ".bonk.yaml".
		viper.AddConfigPath(".")
		viper.SetConfigName(".bonk.yaml")
	}

	viper.AutomaticEnv()

	// If a config file is found, read it in.
	err := viper.ReadInConfig()
	if err == nil {
		slog.Debug("using config file", "file", viper.ConfigFileUsed())
	} else {
		slog.Debug("not using config file", "error", err.Error())
	}
}

func main() {
	slog.SetDefault(
		slog.New(
			slogmulti.
				Pipe(slogctx.NewMiddleware(nil)).
				Handler(pterm.NewSlogHandler(
					pterm.DefaultLogger.
						WithWriter(rootCmd.OutOrStdout()).
						WithLevel(pterm.LogLevelDebug).
						WithTime(false),
				)),
		),
	)

	err := rootCmd.Execute()
	if err != nil {
		slog.Error(err.Error())
		os.Exit(1)
	}
}
