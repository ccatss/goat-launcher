package main

import (
	"fmt"
	"os"

	"github.com/ccatss/goat-launcher/launcher"
	"github.com/spf13/cobra"
)

var runelitePath string

var rootCmd = &cobra.Command{
	Use:   "goat",
	Short: "GoAT is a webkit/webview-free Jagex Launcher",
	Long:  `A webkit and webview-free Jagex Launcher, written in pure Go.`,
	Run: func(cmd *cobra.Command, args []string) {
		var opts []launcher.Option

		if runelitePath != "" {
			opts = append(opts, launcher.WithRuneLite(runelitePath))
		}

		l := launcher.New()
		l.Start()
	},
}

func init() {
	rootCmd.Flags().StringVarP(&runelitePath, "runelite", "r", "", "Path to the runelite launcher")
	rootCmd.AddCommand(loginCmd)
}

func main() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
