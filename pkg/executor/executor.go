package executor

import (
	"fmt"
	"os/exec"
)

type Task struct {
	Workflow  string
	Namespace string
	Name      string
	Command   string
	Args      []string
	Output    string
	Error     error
}

func (t Task) Execute() {
	c := &exec.Cmd{
		Path: t.Command,
		Args: t.Args,
	}

	output, err := c.Output()
	t.Output = string(output)

	if err != nil {
		t.Error = err
	}
	t.Update()
}

func (t Task) Update() {
	fmt.Println(t)
}
