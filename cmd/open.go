package cmd

import (
	"github.com/rekram1-node/pgterm/internal/writer"
	"github.com/spf13/cobra"
)

func NewOpenCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "open",
		Short: "opens UI",
		Long:  "opens the terminal UI in current session",
		Run: func(cmd *cobra.Command, args []string) {
			w := writer.Default()
			// t := termui.New(os.Getenv("PG_URL"))
			_ = w
			// if err := t.Run(); err != nil {
			// 	w.Error(err)
			// }
		},
	}

	return cmd
}

var openCmd = NewOpenCmd()

func init() {
	rootCmd.AddCommand(openCmd)
}
