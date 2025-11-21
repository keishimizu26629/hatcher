package main

import (
	"os"

	"github.com/keisukeshimizu/hatcher/cmd"
)

func main() {
	if err := cmd.Execute(); err != nil {
		os.Exit(1)
	}
}
