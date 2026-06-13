package infra

import (
	"context"
	"errors"
	"fmt"
	"log"
	"os"
	"os/signal"
	"strings"
	"syscall"

	"github.com/alert666/alertmanager-agent/base/conf"
	"github.com/alert666/alertmanager-agent/base/constant"
	baselog "github.com/alert666/alertmanager-agent/base/log"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
	"github.com/spf13/viper"
	"go.uber.org/zap"
)

// NewCmd creates the root cobra command.
func NewCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:           "alertmanager-agent",
		Long:          `alertmanager agent service`,
		SilenceUsage:  true,
		SilenceErrors: true,
		PersistentPreRun: func(cmd *cobra.Command, args []string) {
			serverName := os.Getenv("SERVICE_NAME")
			if serverName != "" {
				viper.Set("server.name", serverName)
				viper.SetEnvPrefix(strings.ToUpper(serverName))
			} else {
				viper.Set("server.name", "alertmanager-agent")
				viper.SetEnvPrefix("ALERTMANAGER_AGENT")
			}
			viper.SetEnvKeyReplacer(strings.NewReplacer(".", "_", "-", "_"))
			viper.AutomaticEnv()
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			return runApp(cmd, args)
		},
	}

	cmd.PersistentFlags().StringP(constant.FlagConfigPath, "c", "./config.yaml", "config file path")
	cmd.PersistentFlags().StringP("log-level", "l", "info", "log level (debug, info, warn, error)")

	bindAllFlagsWithNormalize(cmd.PersistentFlags())
	return cmd
}

func runApp(_ *cobra.Command, _ []string) error {
	cf := viper.GetString("config.path")
	if cf == "" {
		return errors.New("config file path is empty")
	}
	if err := conf.LoadConfig(cf); err != nil {
		return fmt.Errorf("load config failed: %w", err)
	}

	baselog.NewLogger()
	zap.L().Debug("config loaded", zap.Any("config", conf.AllConfig()))

	ctx, stop := signal.NotifyContext(context.TODO(), syscall.SIGINT, syscall.SIGTERM, syscall.SIGQUIT)
	defer stop()

	app, cleanup, err := InitApplication()
	if err != nil {
		return fmt.Errorf("init application failed: %w", err)
	}
	defer cleanup()

	if err := app.Run(ctx); err != nil {
		return fmt.Errorf("run application failed: %w", err)
	}
	zap.L().Info("agent exiting")
	return nil
}

func bindAllFlagsWithNormalize(f *pflag.FlagSet) {
	f.VisitAll(func(flag *pflag.Flag) {
		viperKey := strings.ReplaceAll(flag.Name, "-", ".")
		if err := viper.BindPFlag(viperKey, flag); err != nil {
			log.Fatalf("unable to bind flag %s to viper key %s: %v", flag.Name, viperKey, err)
		}
	})
}
