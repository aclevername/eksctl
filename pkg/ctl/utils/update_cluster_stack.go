package utils

import (
	"os"

	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"

	"github.com/weaveworks/eksctl/pkg/ctl/cmdutils"
)

func updateClusterStackCmd(cmd *cmdutils.Cmd) {
	cmd.SetDescription("update-cluster-stack", "DEPRECATED: Use 'eksctl update cluster' instead", "")

	cmd.CobraCommand.Run = func(cobraCmd *cobra.Command, _ []string) {
		logrus.Errorf(cobraCmd.Short)
		os.Exit(1)
	}
}
