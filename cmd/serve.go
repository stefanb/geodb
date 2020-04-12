package cmd

import (
	api "github.com/autom8ter/geodb/gen/go/geodb"
	"github.com/autom8ter/geodb/server"
	"github.com/autom8ter/geodb/services"
	"github.com/spf13/cobra"
	"log"
)

var serveCmd = &cobra.Command{
	Use:                        "serve",
	Short: "serve starts the geodb server",
	Run: func(cmd *cobra.Command, args []string) {
		s, err := server.NewServer()
		if err != nil {
			log.Fatal(err.Error())
		}
		s.Setup(func(server *server.Server) error {
			api.RegisterGeoDBServer(s.GetGRPCServer(), services.NewGeoDB(s.GetDB(), s.GetStream()))
			return nil
		})
		s.Run()
	},
}