package main

import (
	"log"
	"os"

	"github.com/ostenbom/refunction/funk/funker"
)

func main() {
	f := funker.NewFunker()
	app := f.App()

	err := app.Run(os.Args)
	if err != nil {
		log.Fatal(err)
	}
}
