package exec

import (
	"github.com/arunprasadmudaliar/trinity/pkg/executor"
	"github.com/spf13/cobra"
)

var name string
var namespace string
var kubeconfig string

//Cmd for exec
var Cmd = &cobra.Command{
	Use:   "exec",
	Short: "Executes a workflow",
	Long:  ``,
	//Args:  cobra.ExactArgs(2),
	Run: func(cmd *cobra.Command, args []string) {
		config, _ := cmd.Flags().GetString("kubeconfig")
		executor.Run(config)
	},
}

func init() {
	Cmd.Flags().StringVarP(&name, "name", "n", "", "name of the workflow")
	Cmd.Flags().StringVarP(&namespace, "namespace", "s", "", "namespace of the workflow")
	Cmd.Flags().StringVarP(&kubeconfig, "kubeconfig", "k", "", "path to kubeconfig file")
	Cmd.MarkFlagRequired("name")
	Cmd.MarkFlagRequired("namespace")
}
