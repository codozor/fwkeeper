package cmd

import (
	"github.com/spf13/cobra"
)

func cmdRoot() *cobra.Command {
	cmd := &cobra.Command{
		Use: "fwkeeper",

		Short: "fwkeeper",

		Long: `Port forwarding made easy.`,

		CompletionOptions: cobra.CompletionOptions{HiddenDefaultCmd: true},

		SilenceUsage:  true,
		SilenceErrors: true,
	}

	cmd.AddCommand(cmdStart())

	cmd.PersistentFlags().StringP("config", "c", "fwkeeper.cue", "Configuration file")
	cmd.PersistentFlags().String("kubeconfig", "", "Path to the kubeconfig file (default: $KUBECONFIG or ~/.kube/config)")
	cmd.PersistentFlags().String("context", "", "Kubeconfig context to use (overrides the 'context' field of the configuration)")

	return cmd
}

func Execute() error {
	cmd := cmdRoot()

	return cmd.Execute()
}
