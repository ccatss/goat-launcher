package main

import (
	"errors"
	"fmt"
	"os"
	"path"
	"strings"

	launcher "github.com/ccatss/goat-launcher"
	"github.com/ccatss/goat-launcher/auth"
	"github.com/ccatss/goat-launcher/urlproto"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
	"github.com/spf13/viper"
)

var (
	runelitePath  string
	storeType     string
	storePath     string
	storePassword string
)

var rootCmd = &cobra.Command{
	Use:   "goat",
	Short: "GoAT is a webkit/webview-free Jagex Launcher",
	Long:  `A webkit and webview-free Jagex Launcher, written in pure Go.`,
	PreRunE: func(cmd *cobra.Command, args []string) error {
		return bindFlagsToEnv(cmd)
	},
	RunE: func(cmd *cobra.Command, args []string) error {
		dataDir, err := os.UserConfigDir()

		if err != nil {
			return err
		}

		dataDir = path.Join(dataDir, "goat-launcher")

		if _, err := os.Stat(dataDir); os.IsNotExist(err) {
			err = os.MkdirAll(dataDir, 0755)

			if err != nil {
				return err
			}
		}

		viper.AddConfigPath(dataDir)
		viper.SetConfigName("goat")
		viper.SetConfigType("json")

		if err := viper.ReadInConfig(); err != nil {
			var configFileNotFoundError viper.ConfigFileNotFoundError
			if errors.As(err, &configFileNotFoundError) {
				if writeErr := viper.SafeWriteConfig(); writeErr != nil {
					return err
				}
			} else {
				return fmt.Errorf("unable to read existing config: %v", err)
			}
		}

		opts := []launcher.Option{
			launcher.WithDataDirectory(dataDir),
			launcher.WithConfig(viper.GetViper()),
		}

		if storeType != "" {
			var store auth.Store

			switch storeType {
			case "file":
				store = auth.NewFileStore(storePath, storePassword)
			}

			opts = append(opts, launcher.WithStore(store))
		}

		executable, err := os.Executable()

		if err != nil {
			return fmt.Errorf("unable to determine executable: %v", err)
		}

		if isDefault, err := urlproto.IsDefaultHandler("jagex", executable); err != nil || !isDefault {
			// Set the launcher to use a prompt for the jagex: url instead
			opts = append(opts, launcher.PromptForCode())
		}

		l, err := launcher.New(opts...)

		if err != nil {
			return fmt.Errorf("launcher creation failed: %v", err)
		}

		l.Start()

		return nil
	},
}

func init() {
	rootCmd.Flags().StringP("runelite-path", "r", "", "Path to the runelite launcher")
	rootCmd.Flags().StringVarP(&storeType, "store", "k", "", "Store backend")
	rootCmd.Flags().StringVarP(&storePath, "store-path", "f", "", "Store path")
	rootCmd.Flags().StringVarP(&storePassword, "store-password", "p", "", "Store password")
	rootCmd.AddCommand(loginCmd)
}

func bindFlagsToEnv(cmd *cobra.Command) error {
	viper.AutomaticEnv()

	viper.SetEnvKeyReplacer(strings.NewReplacer("-", "_"))

	var bindErr error

	cmd.Flags().VisitAll(func(flag *pflag.Flag) {
		// If an environment variable exists, it will map to APP_FLAGNAME
		err := viper.BindPFlag(flag.Name, flag)
		if err != nil {
			bindErr = err
		}
	})

	return bindErr
}

func main() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
