package main

import (
	"log"
	"myblog_last_new/internal/app"
)

func main() {
	if err := app.Run(); err != nil {
		log.Fatal(err)
	}
}
