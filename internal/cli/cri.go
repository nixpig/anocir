package cli

import (
	"errors"
	"fmt"
	"io/fs"
	"net"
	"os"
	"os/signal"
	"syscall"

	"github.com/nixpig/anocir/internal/cri"
	"github.com/spf13/cobra"
)

func criCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "cri [flags]",
		Short:   "Start a CRI server",
		Example: "  anocir cri --socket /var/run/anocir.sock",
		Args:    cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			socket, _ := cmd.Flags().GetString("socket")
			if socket == "" {
				return errors.New("socket cannot be empty")
			}

			if err := os.Remove(socket); err != nil &&
				!errors.Is(err, fs.ErrNotExist) {
				return fmt.Errorf("failed to remove existing socket: %w", err)
			}

			listener, err := net.Listen("unix", socket)
			if err != nil {
				return fmt.Errorf("failed to listen on socket: %w", err)
			}

			server := cri.NewCRIServer()

			errCh := make(chan error, 1)
			go func() {
				cmd.OutOrStdout().Write(
					fmt.Appendf(nil, "starting server: %s\n", listener.Addr().String()),
				)

				errCh <- server.Start(listener)
			}()

			ctx, cancel := signal.NotifyContext(
				cmd.Context(),
				syscall.SIGTERM,
				os.Interrupt,
			)
			defer cancel()

			select {
			case err := <-errCh:
				if err != nil {
					return fmt.Errorf("server stopped with error: %w", err)
				}
			case <-ctx.Done():
				cmd.OutOrStdout().Write([]byte("shutting down server\n"))
				server.Shutdown()
				_ = <-errCh
			}

			return nil
		},
	}

	cmd.Flags().
		StringP("socket", "s", "/var/run/anocir.sock", "UNIX socket for the CRI server")

	return cmd
}
