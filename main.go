package main

import (
	"github.com/autom8ter/geodb/cmd"
	log "github.com/sirupsen/logrus"
)

func main() {
	if err := cmd.Execute(); err != nil {
		log.Fatal(err.Error())
	}
}
