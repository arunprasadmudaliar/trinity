package cmd

import (
	"os"

	"github.com/arunprasadmudaliar/trinity/cmd/version"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
)

// rootCmd represents the base command when called without any subcommands
var rootCmd = &cobra.Command{
	Use:   "trinity",
	Short: "Create Business workflows using Trinity",
	Long:  ``,
}

//Execute func
func Execute() {
	if err := rootCmd.Execute(); err != nil {
		log.Panicf("Failed to start:%v", err)
		os.Exit(1)
	}
}

func init() {
	rootCmd.AddCommand(version.Cmd)
}
