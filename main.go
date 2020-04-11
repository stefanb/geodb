package main

import (
	api "github.com/autom8ter/geodb/gen/go/geodb"
	"github.com/autom8ter/geodb/server"
	"github.com/autom8ter/geodb/services"
	log "github.com/sirupsen/logrus"
)

func main() {
	s, err := server.NewServer()
	if err != nil {
		log.Fatal(err.Error())
	}
	s.Setup(func(server *server.Server) error {
		api.RegisterGeoDBServer(s.GetGRPCServer(), services.NewGeoDB(s.GetDB(), s.GetStream()))
		return nil
	})
	s.Run()
}
