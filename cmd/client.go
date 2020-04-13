package cmd

import (
	"context"
	"fmt"
	api "github.com/autom8ter/geodb/gen/go/geodb"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"google.golang.org/grpc"
)

func init() {
	setCmd.Flags().StringVar(&target, "target", "localhost:8080", "target server url")
	setCmd.Flags().StringVar(&key, "key", "", "object key")
	setCmd.Flags().StringSliceVar(&keys, "keys", []string{"*"}, "object keys")
	setCmd.Flags().Float64Var(&lat, "lat", 0, "latitude")
	setCmd.Flags().Float64Var(&lon, "lon", 0, "longitude")
	setCmd.Flags().Int64Var(&radius, "rad", 50, "radius")
}

var (
	target string
	key    string
	lat    float64
	lon    float64
	radius int64
	keys   []string
)

var setCmd = &cobra.Command{
	Use:   "set",
	Short: "set an object",
	Run: func(cmd *cobra.Command, args []string) {
		conn, err := grpc.Dial(target, grpc.WithInsecure())
		if err != nil {
			log.Fatal(err.Error())
		}
		client := api.NewGeoDBClient(conn)
		resp, err := client.SetObject(context.Background(), &api.SetObjectRequest{
			Object: map[string]*api.Object{
				key: &api.Object{
					Point: &api.Point{
						Lat: lat,
						Lon: lon,
					},
					Radius: radius,
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

var getCmd = &cobra.Command{
	Use:   "get",
	Short: "get an object",
	Run: func(cmd *cobra.Command, args []string) {
		conn, err := grpc.Dial(target, grpc.WithInsecure())
		if err != nil {
			log.Fatal(err.Error())
		}
		client := api.NewGeoDBClient(conn)
		resp, err := client.GetObject(context.Background(), &api.GetObjectRequest{
			Keys: keys,
		})
		if err != nil {
			log.Fatal(err.Error())
		}
		fmt.Println(resp.String())
	},
}

var streamCmd = &cobra.Command{
	Use:   "stream",
	Short: "stream object updates",
	Run: func(cmd *cobra.Command, args []string) {
		conn, err := grpc.Dial(target, grpc.WithInsecure())
		if err != nil {
			log.Fatal(err.Error())
		}
		client := api.NewGeoDBClient(conn)
		resp, err := client.StreamObject(context.Background(), &api.StreamObjectRequest{})
		if err != nil {
			log.Fatal(err.Error())
		}
		for {
			res, err := resp.Recv()
			if err != nil {
				log.Error(err.Error())
			}
			fmt.Println(res.String())
		}
	},
}
