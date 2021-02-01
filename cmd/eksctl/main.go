package main

import (
	"fmt"
	"github.com/kris-nova/logger"
	"github.com/spf13/cobra"
	"github.com/weaveworks/eksctl/pkg/ctl/set"
	"github.com/weaveworks/eksctl/pkg/ctl/unset"
	"github.com/weaveworks/eksctl/pkg/ctl/upgrade"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"os"
	"time"

	"github.com/weaveworks/eksctl/pkg/ctl/cmdutils"
	"github.com/weaveworks/eksctl/pkg/ctl/completion"
	"github.com/weaveworks/eksctl/pkg/ctl/create"
	"github.com/weaveworks/eksctl/pkg/ctl/delete"
	"github.com/weaveworks/eksctl/pkg/ctl/drain"
	"github.com/weaveworks/eksctl/pkg/ctl/enable"
	"github.com/weaveworks/eksctl/pkg/ctl/generate"
	"github.com/weaveworks/eksctl/pkg/ctl/get"
	"github.com/weaveworks/eksctl/pkg/ctl/scale"
	"github.com/weaveworks/eksctl/pkg/ctl/update"
	"github.com/weaveworks/eksctl/pkg/ctl/utils"
)

func addCommands(rootCmd *cobra.Command, flagGrouping *cmdutils.FlagGrouping) {
	rootCmd.AddCommand(create.Command(flagGrouping))
	rootCmd.AddCommand(get.Command(flagGrouping))
	rootCmd.AddCommand(update.Command(flagGrouping))
	rootCmd.AddCommand(upgrade.Command(flagGrouping))
	rootCmd.AddCommand(delete.Command(flagGrouping))
	rootCmd.AddCommand(set.Command(flagGrouping))
	rootCmd.AddCommand(unset.Command(flagGrouping))
	rootCmd.AddCommand(scale.Command(flagGrouping))
	rootCmd.AddCommand(drain.Command(flagGrouping))
	rootCmd.AddCommand(generate.Command(flagGrouping))
	rootCmd.AddCommand(enable.Command(flagGrouping))
	rootCmd.AddCommand(utils.Command(flagGrouping))
	rootCmd.AddCommand(completion.Command(rootCmd))

	cmdutils.AddResourceCmd(flagGrouping, rootCmd, versionCmd)
}

func SyslogTimeEncoder(t time.Time, enc zapcore.PrimitiveArrayEncoder) {
	enc.AppendString(t.Format("10-04-19 12:00:17"))
}

func initZapLog() *zap.Logger {
	config := zap.NewDevelopmentConfig()
	config.EncoderConfig.EncodeLevel = zapcore.CapitalColorLevelEncoder
	config.EncoderConfig.TimeKey = "timestamp"
	config.EncoderConfig.EncodeTime = SyslogTimeEncoder
	config.DisableCaller = true
	config.DisableStacktrace = true
	
	logger, _ := config.Build()
	return logger
}


func init()  {
	loggerMgr := initZapLog()
	zap.ReplaceGlobals(loggerMgr)
	defer loggerMgr.Sync() // flushes buffer, if any

	zap.S().Infof(`eksctl version 0.36.2`)
	zap.S().Infof(`using region us-west-2`)
	zap.S().Infof(`1 existing iamserviceaccount(s) (baz/foo) will be excluded`)
	zap.S().Infof(`1 iamserviceaccount (baz/fooo) was included (based on the include/exclude rules)`)
	zap.S().Warnf("serviceaccounts that exists in Kubernetes will be excluded, use --override-existing-serviceaccounts to override")
	zap.S().Infof(`1 task: { 2 sequential sub-tasks: { create IAM role for serviceaccount "baz/fooo", create serviceaccount "baz/fooo" } }`)
	zap.S().Infof(`building iamserviceaccount stack "eksctl-jk-addon-iamserviceaccount-baz-fooo"`)
	zap.S().Infof(`deploying stack "eksctl-jk-addon-iamserviceaccount-baz-fooo"`)
	zap.S().Infof(`waiting for CloudFormation stack "eksctl-jk-addon-iamserviceaccount-baz-fooo"`)
	zap.S().Infof(`waiting for CloudFormation stack "eksctl-jk-addon-iamserviceaccount-baz-fooo"`)
	zap.S().Infof(`waiting for CloudFormation stack "eksctl-jk-addon-iamserviceaccount-baz-fooo"`)
	zap.S().Infof(`created serviceaccount "baz/fooo"`)

os.Exit(1)
}

func main() {
	rootCmd := &cobra.Command{
		Use:   "eksctl [command]",
		Short: "The official CLI for Amazon EKS",
		Run: func(c *cobra.Command, _ []string) {
			if err := c.Help(); err != nil {
				logger.Debug("ignoring cobra error %q", err.Error())
			}
		},
		SilenceUsage: true,
	}

	flagGrouping := cmdutils.NewGrouping()

	addCommands(rootCmd, flagGrouping)
	checkCommand(rootCmd)

	rootCmd.PersistentFlags().BoolP("help", "h", false, "help for this command")
	rootCmd.PersistentFlags().IntVarP(&logger.Level, "verbose", "v", 3, "set log level, use 0 to silence, 4 for debugging and 5 for debugging with AWS debug logging")

	colorValue := rootCmd.PersistentFlags().StringP("color", "C", "true", "toggle colorized logs (valid options: true, false, fabulous)")

	cobra.OnInitialize(func() {
		// Control colored output
		logger.Color = *colorValue == "true"
		logger.Fabulous = *colorValue == "fabulous"
		// Add timestamps for debugging
		logger.Timestamps = logger.Level >= 4
	})

	rootCmd.SetUsageFunc(flagGrouping.Usage)

	if err := rootCmd.Execute(); err != nil {
		os.Exit(1)
	}
}

func checkCommand(rootCmd *cobra.Command) {
	for _, cmd := range rootCmd.Commands() {
		// just a precaution as the verb command didn't have runE
		if cmd.RunE != nil {
			continue
		}
		cmd.RunE = func(c *cobra.Command, args []string) error {
			var e error
			if len(args) == 0 {
				e = fmt.Errorf("please provide a valid resource for \"%s\"", c.Name())
			} else {
				e = fmt.Errorf("unknown resource type \"%s\"", args[0])
			}
			fmt.Printf("Error: %s\n\n", e.Error())

			if err := c.Help(); err != nil {
				logger.Debug("ignoring cobra error %q", err.Error())
			}
			return e
		}
	}
}
