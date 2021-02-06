package runner

const registry = "arunmudaliar/"
const bash = "bash:latest"
const az = "azure:latest"
const git = "git:latest"
const aws = "aws:latest"

func getImage(name string) string {
	switch name {
	case "sh":
		return registry + bash
	case "git":
		return registry + git
	case "az":
		return registry + az
	case "aws":
		return registry + aws
	default:
		return registry + bash
	}
}
