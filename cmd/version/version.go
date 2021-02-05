package version

import (
	"fmt"

	"github.com/spf13/cobra"
)

//Cmd for version number
var Cmd = &cobra.Command{
	Use:   "version",
	Short: "Display version number of trinity",
	Long:  ``,
	//Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Println("Trinity version:1.0.0")
	},
}
