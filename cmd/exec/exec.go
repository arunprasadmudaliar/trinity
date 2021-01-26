package exec

import (
	"strings"

	"github.com/arunprasadmudaliar/trinity/pkg/executor"
	"github.com/spf13/cobra"
)

var workflow string
var task string
var namespace string
var command string
var args string

//Cmd for exec
var Cmd = &cobra.Command{
	Use:   "exec",
	Short: "Executes a task in a workflow",
	Long:  ``,
	//Args:  cobra.ExactArgs(2),
	Run: func(cmd *cobra.Command, args []string) {
		wf, _ := cmd.Flags().GetString("workflow")
		task, _ := cmd.Flags().GetString("task")
		ns, _ := cmd.Flags().GetString("namespace")
		command, _ := cmd.Flags().GetString("command")
		cmdArgs, _ := cmd.Flags().GetString("args")

		e := executor.Task{
			Workflow:  wf,
			Namespace: ns,
			Name:      task,
			Command:   command,
			Args:      strings.Split(cmdArgs, " "),
		}
		e.Execute()
	},
}

func init() {
	Cmd.Flags().StringVarP(&command, "command", "c", "", "complete command to be executed")
	Cmd.Flags().StringVarP(&args, "args", "a", "", "space separated argument list")
	Cmd.Flags().StringVarP(&workflow, "workflow", "w", "", "name of the workflow")
	Cmd.Flags().StringVarP(&namespace, "namespace", "n", "", "namespace of the workflow")
	Cmd.Flags().StringVarP(&task, "task", "t", "", "name of the task")
	//Cmd.Flags().StringVarP(&args, "args", "a", "", "arguments to pass to the command, separated by space")
	Cmd.MarkFlagRequired("command")
	Cmd.MarkFlagRequired("workflow")
	Cmd.MarkFlagRequired("namespace")
	Cmd.MarkFlagRequired("task")
	//Cmd.MarkFlagRequired("args")
}
