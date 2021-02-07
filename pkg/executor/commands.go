package executor

const az = "az"
const aws = "aws"
const bash = "/bin/bash"
const git = "git"

func getCmd(name string) string {
	switch name {
	case "bash":
		return bash
	case "git":
		return git
	case "az":
		return az
	case "aws":
		return aws
	default:
		return bash
	}
}

func getArgs(name string, args []string) []string {
	if len(args) > 0 {
		return args
	}
	switch name {
	case "bash":
		return []string{"-lc", "uname", "-a"}
	case "git":
		return []string{"version"}
	case "az":
		return []string{"version"}
	case "aws":
		return []string{"--version"}
	default:
		return []string{"-lc", "uname", "-a"}
	}
}
