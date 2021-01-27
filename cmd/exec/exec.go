package exec

import (
	"github.com/arunprasadmudaliar/trinity/pkg/executor"
	"github.com/spf13/cobra"
)

var workflow string
var id int
var namespace string
var kubeconfig string

//Cmd for exec
var Cmd = &cobra.Command{
	Use:   "exec",
	Short: "Executes a task in a workflow",
	Long:  ``,
	//Args:  cobra.ExactArgs(2),
	Run: func(cmd *cobra.Command, args []string) {
		wf, _ := cmd.Flags().GetString("workflow")
		ns, _ := cmd.Flags().GetString("namespace")
		config, _ := cmd.Flags().GetString("kubeconfig")
		id, _ := cmd.Flags().GetInt("id")

		executor.Execute(config, wf, ns, id)

	},
}

func init() {
	Cmd.Flags().StringVarP(&kubeconfig, "kubeconfig", "k", "", "path to kubeconfig file")
	Cmd.Flags().StringVarP(&workflow, "workflow", "w", "", "name of the workflow")
	Cmd.Flags().StringVarP(&namespace, "namespace", "s", "", "namespace of the workflow")
	Cmd.Flags().IntVarP(&id, "id", "i", 0, "task id")
	Cmd.MarkFlagRequired("workflow")
	Cmd.MarkFlagRequired("namespace")
	Cmd.MarkFlagRequired("id")
}
