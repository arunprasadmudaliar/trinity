package cmd

import (
	"os"

	"github.com/arunprasadmudaliar/trinity/cmd/ctrl"
	"github.com/arunprasadmudaliar/trinity/cmd/exec"
	"github.com/arunprasadmudaliar/trinity/cmd/version"
	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
)

// rootCmd represents the base command when called without any subcommands
var rootCmd = &cobra.Command{
	Use:   "ty",
	Short: "Create Business workflows using Trinity",
	Long:  ``,
}

//Execute func
func Execute() {
	if err := rootCmd.Execute(); err != nil {
		logrus.WithError(err).Error("An error occurred")
		os.Exit(1)
	}
}

func init() {
	rootCmd.AddCommand(version.Cmd)
	rootCmd.AddCommand(ctrl.Cmd)
	rootCmd.AddCommand(exec.Cmd)
}
