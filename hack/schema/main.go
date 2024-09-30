package main

import (
	"os/exec"
)

func main() {
	cmd := exec.Command("./hack/schema/create-schema.sh")
	_, err := cmd.Output()
	if err != nil {
		panic(err)
	}
}
