package command

import "os/exec"

func Run(cmdString string) (string, error) {
	cmd := exec.Command("sh", "-c", cmdString)
	output, err := cmd.CombinedOutput()
	return string(output), err
}
