package cmd

import (
	"context"
	"fmt"
	api "github.com/autom8ter/geodb/gen/go/geodb"
	"github.com/spf13/cobra"
	"google.golang.org/grpc"
	"log"
)

func init() {
	setCmd.Flags().StringVarP(&target, "target", "t", "localhost:8080", "target server url")
	setCmd.Flags().StringVarP(&key, "key", "k", "", "object key")
	setCmd.Flags().Float64Var(&lat, "lat", 0, "latitude")
	setCmd.Flags().Float64Var(&lon, "lon", 0, "longitude")
	setCmd.Flags().Int64Var(&radius, "rad", 50, "radius")
}

var (
	target string
	key string
	lat float64
	lon float64
	radius int64
)

var setCmd = &cobra.Command{
	Use:                        "set",
	Short: "set an object",
	Run: func(cmd *cobra.Command, args []string) {
		conn, err := grpc.Dial(target, grpc.WithInsecure())
		if err != nil {
			log.Fatal(err.Error())
		}
		client := api.NewGeoDBClient(conn)
		resp, err := client.Set(context.Background(), &api.SetRequest{
			Object: map[string]*api.Object{
				key: &api.Object{
					Point:                &api.Point{
						Lat:                  lat,
						Lon:                  lon,
					},
					Radius:               radius,
					Metadata: map[string]string{
						"testing": "true",
					},
				},
			},
		})
		if err != nil {
			log.Fatal(err.Error())
		}
		fmt.Println(resp.String())
	},
}
