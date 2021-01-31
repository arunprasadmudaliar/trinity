package exec

import (
	"github.com/arunprasadmudaliar/trinity/pkg/executor"
	"github.com/spf13/cobra"
)

var workflow string
var runid int
var taskid int
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
		runid, _ := cmd.Flags().GetInt("runid")
		taskid, _ := cmd.Flags().GetInt("taskid")

		executor.Execute(config, wf, ns, runid, taskid)

	},
}

func init() {
	Cmd.Flags().StringVarP(&kubeconfig, "kubeconfig", "k", "", "path to kubeconfig file")
	Cmd.Flags().StringVarP(&workflow, "workflow", "w", "", "name of the workflow")
	Cmd.Flags().StringVarP(&namespace, "namespace", "s", "", "namespace of the workflow")
	Cmd.Flags().IntVarP(&runid, "runid", "r", 0, "run id")
	Cmd.Flags().IntVarP(&taskid, "taskid", "t", 0, "task id")
	Cmd.MarkFlagRequired("workflow")
	Cmd.MarkFlagRequired("namespace")
	Cmd.MarkFlagRequired("runid")
	Cmd.MarkFlagRequired("taskid")
}
