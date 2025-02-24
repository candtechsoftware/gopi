package main

import (
	"fmt"
	"os"

	"percipio.com/gopi/lib/app"
)

func main() {
	application, err := app.New()
	if err != nil {
		fmt.Fprintln(os.Stderr, "Error:", err)
		os.Exit(1)
	}
	application.Run()
}
