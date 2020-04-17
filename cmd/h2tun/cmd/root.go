/*
Copyright © 2020 QiuYuzhou <charlie@shundaojia.com>

*/
package cmd

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"syscall"

	"github.com/spf13/cobra"
	"go.uber.org/zap"

	"github.com/qiuyuzhou/h2tun/internal/app/client"
	"github.com/qiuyuzhou/h2tun/internal/app/server"
	"github.com/qiuyuzhou/h2tun/internal/pkg/env"
)

var version = "undefined"

var isDebug bool
var fastOpen bool
var useTLSInClient bool
var inServerMode bool
var tunnelPath string

var localHost string
var localPort uint16

var remoteHost string
var remotePort uint16

var keyFile string
var certFile string

var logger *zap.Logger

// rootCmd represents the base command when called without any subcommands
var rootCmd = &cobra.Command{
	Use:   "h2tun",
	Short: "A brief description of your application",
	// 	Long: `A longer description that spans multiple lines and likely contains
	// examples and usage of using your application. For example:

	// Cobra is a CLI library for Go that empowers applications.
	// This application is a tool to generate the needed files
	// to quickly create a Cobra application.`,
	// Uncomment the following line if your bare application
	// has an action associated with it:
	Run: func(cmd *cobra.Command, args []string) {
		if inServerMode {
			logger.Info("Run in server mode...")

			ctx, cancel := context.WithCancel(context.Background())

			signals := make(chan os.Signal, 1)
			signal.Notify(signals, syscall.SIGTERM, syscall.SIGINT)

			go func() {
				_ = <-signals
				signal.Stop(signals)
				cancel()
			}()

			plugin := &server.Plugin{
				Logger:   logger,
				FromAddr: env.ConcatHostPort(remoteHost, remotePort),
				ToAddr:   env.ConcatHostPort(localHost, localPort),
				Path:     tunnelPath,
				KeyFile:  keyFile,
				CertFile: certFile,
			}

			err := plugin.Serve(ctx)
			if err != nil && err != http.ErrServerClosed {
				logger.Error("plugin shutdown with error", zap.Error(err))
			}

		} else {
			logger.Info("Run in client mode...")

			ctx, cancel := context.WithCancel(context.Background())

			signals := make(chan os.Signal, 1)
			signal.Notify(signals, syscall.SIGTERM, syscall.SIGINT)

			go func() {
				_ = <-signals
				signal.Stop(signals)
				cancel()
			}()

			plugin := &client.Plugin{
				Logger:   logger,
				FromAddr: env.ConcatHostPort(localHost, localPort),
				ToAddr:   env.ConcatHostPort(remoteHost, remotePort),
				Path:     tunnelPath,
				UseTLS:   useTLSInClient,
			}

			err := plugin.Serve(ctx)
			if err != nil {
				logger.Error("plugin shutdown with error", zap.Error(err))
			}
		}
		logger.Sync()
	},
}

// Execute adds all child commands to the root command and sets flags appropriately.
// This is called by main.main(). It only needs to happen once to the rootCmd.
func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}

func init() {
	rootCmd.Version = version

	logger = zap.NewNop()
	cobra.OnInitialize(initLogger)
	cobra.OnInitialize(overideFromEnv)

	rootCmd.PersistentFlags().BoolVarP(&inServerMode, "server", "s", false, "Run in server mode")
	rootCmd.PersistentFlags().StringVarP(&tunnelPath, "path", "p", "/h2tunnel", "Handle tunnel at the path")

	rootCmd.PersistentFlags().StringVar(&localHost, "local-host", "127.0.0.1", "")
	rootCmd.PersistentFlags().Uint16Var(&localPort, "local-port", 18086, "")
	rootCmd.PersistentFlags().StringVar(&remoteHost, "remote-host", "127.0.0.1", "")
	rootCmd.PersistentFlags().Uint16Var(&remotePort, "remote-port", 18096, "")
	rootCmd.PersistentFlags().BoolVar(&fastOpen, "fast-open", false, "Enable TCP fast open.")
	rootCmd.PersistentFlags().BoolVar(&isDebug, "debug", false, "Enable debug mode.")

	// Cobra also supports local flags, which will only run
	// when this action is called directly.
	// rootCmd.Flags().BoolP("toggle", "t", false, "Help message for toggle")
}

func initLogger() {
	if isDebug {
		config := zap.NewDevelopmentConfig()
		_logger, err := config.Build()
		if err != nil {
			panic("failed to initial logger")
		}
		logger = _logger
	} else {
		config := zap.NewProductionConfig()
		config.Encoding = "console"
		_logger, err := config.Build()
		if err != nil {
			panic("failed to initial logger")
		}
		logger = _logger
	}
}

func overideFromEnv() {
	args, err := env.ParseEnv()
	if err != nil {
		logger.Fatal("failed to parse arguments from env", zap.Error(err))
	}

	if v, ok := args.Get("localHost"); ok {
		localHost = v
	}
	if v, ok := args.Get("localPort"); ok {
		p, err := strconv.ParseUint(v, 10, 16)
		if err != nil {
			logger.Fatal("failed to parse SS_LOCAL_PORT", zap.String("SS_LOCAL_PORT", v))
		}
		localPort = uint16(p)
	}

	if v, ok := args.Get("remoteHost"); ok {
		remoteHost = v
	}
	if v, ok := args.Get("remotePort"); ok {
		p, err := strconv.ParseUint(v, 10, 16)
		if err != nil {
			logger.Fatal("failed to parse SS_REMOTE_PORT", zap.String("SS_REMOTE_PORT", v))
		}
		remotePort = uint16(p)
	}

	if _, ok := args.Get("tls"); ok {
		useTLSInClient = true
	}

	if _, ok := args.Get("server"); ok {
		inServerMode = true
	}

	if _, ok := args.Get("debug"); ok {
		isDebug = true
	}

	if v, ok := args.Get("path"); ok {
		tunnelPath = v
	}

	if v, ok := args.Get("keyFile"); ok {
		keyFile = v
	}

	if v, ok := args.Get("certFile"); ok {
		certFile = v
	}
}
