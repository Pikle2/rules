package main

import (
	"fmt"
	"log"
	"os/exec"

	"github.com/Pikle2/rules/cli/commands"
)

func main() {
	commands.Execute()
	out, err := exec.Command(" battlesnake play -W 11 -H 11 --name soup --url http://0.0.0.0:8000 -g -solo --browser").Output()
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("test %s\n", out)
}
