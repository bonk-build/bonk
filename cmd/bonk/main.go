// Copyright © 2025 Colden Cullen
// SPDX-License-Identifier: MIT

// bonk runs a test build operation.
package main

import (
	"context"
	"log/slog"
	"os"
	"path"

	"github.com/charmbracelet/fang"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"

	"go.bonk.build/pkg/driver"
	"go.bonk.build/pkg/observer/bubbletea"
	"go.bonk.build/pkg/task"
)

var (
	cfgFile     string
	concurrency int
)

// rootCmd represents the base command when called without any subcommands.
var rootCmd = &cobra.Command{
	Use:   "bonk",
	Short: "A cue-based configuration build system.",

	RunE: func(cmd *cobra.Command, _ []string) error {
		cwd, err := os.Getwd()
		if err != nil {
			return err //nolint:wrapcheck
		}

		bubble := bubbletea.New(cmd.Context(), true)

		err = driver.Run(cmd.Context(), driver.MakeDefaultOptions().
			WithConcurrency(concurrency).
			WithObservers(bubble.OnTaskStatusMsg).
			WithPlugins(
				"go.bonk.build/plugins/test",
				"go.bonk.build/plugins/k8s/resources",
				"go.bonk.build/plugins/k8s/kustomize",
			).
			WithLocalSession(path.Join(cwd, "testdata"),
				task.New(
					task.NewID("Test", "Test"),
					"test.Test",
					map[string]any{
						"value": 3,
					},
				),
				task.New(
					task.NewID("Test", "Resources"),
					"resources.Resources",
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
				),
				task.New(
					task.NewID("Test", "Kustomize"),
					"kustomize.Kustomize",
					map[string]any{},
					task.WithInputs(
						".bonk/Test.Resources/resources.yaml",
					),
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
		StringVarP(&cfgFile, "config", "c", "", "config file (default is .bonk.yaml)")
	rootCmd.PersistentFlags().
		IntVarP(&concurrency, "concurrency", "j", 100, "The max number of goroutines to run (negative for no limit)")

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
	err := fang.Execute(context.Background(), rootCmd)
	if err != nil {
		slog.Error(err.Error())
		os.Exit(1)
	}
}
