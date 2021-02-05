package run

import (
	"github.com/arunprasadmudaliar/trinity/pkg/runner"
	"github.com/spf13/cobra"
)

var name string
var namespace string
var kubeconfig string

//Cmd for exec
var Cmd = &cobra.Command{
	Use:   "run",
	Short: "starts a workflow",
	Long:  ``,
	//Args:  cobra.ExactArgs(2),
	Run: func(cmd *cobra.Command, args []string) {
		config, _ := cmd.Flags().GetString("kubeconfig")
		name, _ := cmd.Flags().GetString("name")
		ns, _ := cmd.Flags().GetString("namespace")
		runner.Run(config, name, ns)
	},
}

func init() {
	Cmd.Flags().StringVarP(&name, "name", "w", "", "name of the workflow")
	Cmd.Flags().StringVarP(&namespace, "namespace", "n", "", "namespace of the workflow")
	Cmd.Flags().StringVarP(&kubeconfig, "kubeconfig", "k", "", "path to kubeconfig file")
	Cmd.MarkFlagRequired("name")
	Cmd.MarkFlagRequired("namespace")
}
