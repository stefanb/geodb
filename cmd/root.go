package cmd

import (
	"github.com/spf13/cobra"
)

func init() {
	rootCmd.AddCommand(serveCmd, setCmd, getCmd, streamCmd, streamEventsCmd)
}

var rootCmd = &cobra.Command{
	Use:  "geodb",
	Long: "geodb is a persistant geospatial database written in Go",
}

func Execute() error {
	return rootCmd.Execute()
}
