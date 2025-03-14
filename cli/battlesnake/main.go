package main

import (
	"fmt"
	"log"
	"os/exec"

	"github.com/Pikle2/rules/cli/commands"
)

func main() {
	commands.Execute()

	cmd := exec.Command(
		"battlesnake",
		"play",
		"-W", "11",
		"-H", "11",
		"--name", "soup",
		"--url", "http://0.0.0.0:8000",
		"-g",
		"-solo",
		"--browser",
	)

	out, err := cmd.CombinedOutput()
	if err != nil {
		log.Fatalf("command failed: %v\nOutput: %s", err, out)
	}
	fmt.Printf("Output: %s\n", out)
}
