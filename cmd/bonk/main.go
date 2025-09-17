// Copyright © 2025 Colden Cullen
// SPDX-License-Identifier: MIT

package main

import (
	"log/slog"
	"os"
	"path"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"

	"go.bonk.build/pkg/driver"
	"go.bonk.build/pkg/driver/basic"
	"go.bonk.build/pkg/scheduler/bubbletea"
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
		cwd, err := os.Getwd()
		cobra.CheckErr(err)

		drv, err := basic.New(cmd.Context(), bubbletea.New(false),
			driver.WithPlugins(
				"go.bonk.build/plugins/test",
				"go.bonk.build/plugins/k8s/resources",
				"go.bonk.build/plugins/k8s/kustomize",
			),
			driver.WithLocalSession(path.Join(cwd, "testdata"),
				driver.WithTask(
					"test.Test",
					"Test.Test",
					map[string]any{
						"value": 3,
					},
				),
				driver.WithTask(
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
				),
				driver.WithTask(
					"kustomize.Kustomize",
					"Test.Kustomize",
					map[string]any{},
					".bonk/Test.Resources/resources.yaml",
				),
			),
		)
		cobra.CheckErr(err)
		defer drv.Shutdown(cmd.Context())

		drv.Run()
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
	err := rootCmd.Execute()
	if err != nil {
		slog.Error(err.Error())
		os.Exit(1)
	}
}
