package ctrl

import (
	"github.com/arunprasadmudaliar/trinity/pkg/controller"
	"github.com/spf13/cobra"
)

var kubeconfig string

//Cmd for version number
var Cmd = &cobra.Command{
	Use:   "ctrl",
	Short: "starts the controller",
	Long:  ``,
	//Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		config, _ := cmd.Flags().GetString("kubeconfig")
		controller.Start(config)
	},
}

func init() {
	Cmd.Flags().StringVarP(&kubeconfig, "kubeconfig", "k", "", "path to kubeconfig file")
}
