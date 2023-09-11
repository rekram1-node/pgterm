package cmd

import (
	"os"

	"github.com/spf13/cobra"
)

var rootCmd = &cobra.Command{
	Use:   "pgterm",
	Short: "pgterm is a terminal UI for psql databases",
	Long:  `Terminal UI made to be a opinionated, lighter, alternative to PgAdmin`,
	Run: func(cmd *cobra.Command, args []string) {
		if cmd == cmd.Root() {
			_ = cmd.Help()
		}
	},
}

func Execute() {
	if err := rootCmd.Execute(); err != nil {
		os.Exit(1)
	}
}
