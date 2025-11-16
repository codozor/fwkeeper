package cmd

import (
	"os"
	"os/signal"
	"syscall"

	"github.com/spf13/cobra"

	"github.com/samber/do/v2"

	"github.com/codozor/fwkeeper/internal/app"
	"github.com/codozor/fwkeeper/internal/bootstrap"
	"github.com/codozor/fwkeeper/internal/config"
)

func cmdStart() *cobra.Command {

	cmd := &cobra.Command{
		Use:   "run",
		Short: "run fwkeeper",
		Long:  `Run fwkeeper in interactive mode.`,

		RunE: func(cmd *cobra.Command, args []string) error {
			cfgFilename := cmd.Flag("config").Value.String()

			configuration, err := config.ReadConfiguration(cfgFilename)
			if err != nil {
				return err
			}

			injector := do.New()

			// Provide configuration to the injector
			do.ProvideValue(injector, configuration)

			// Bootstrap all dependencies
			bootstrap.Package(injector)

			runner, err := do.Invoke[*app.Runner](injector)
			if err != nil {
				return err
			}

			if err := runner.Start(); err != nil {
				return err
			}

			// Setup signal handler for graceful shutdown
			sigCh := make(chan os.Signal, 1)
			signal.Notify(sigCh, os.Interrupt, syscall.SIGTERM)
			<-sigCh

			runner.Shutdown()

			return nil
		},
	}

	return cmd
}
