package cri

import (
	"errors"
	"fmt"
	"io/fs"
	"net"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"

	"github.com/spf13/cobra"
)

func Cmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "anocird [flags]",
		Short:   "Start an anocir CRI server",
		Example: "  anocird --socket /run/anocir/anocird.sock",
		Args:    cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			socket, _ := cmd.Flags().GetString("socket")
			if socket == "" {
				return errors.New("socket cannot be empty")
			}

			listener, err := setupListener(socket)
			if err != nil {
				return fmt.Errorf("failed to setup socket: %w", err)
			}

			server := newCRIServer(listener)

			errCh := make(chan error, 1)
			go func() {
				fmt.Fprintln(cmd.OutOrStdout(), "starting server")
				errCh <- server.start()
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
				fmt.Fprintln(cmd.OutOrStdout(), "shutting down server")
				server.shutdown()
				<-errCh
			}

			return nil
		},
	}

	cmd.Flags().
		StringP("socket", "s", "/run/anocir/anocird.sock", "UNIX socket for the CRI server")

	return cmd
}

func setupListener(socket string) (net.Listener, error) {
	if err := os.Remove(socket); err != nil &&
		!errors.Is(err, fs.ErrNotExist) {
		return nil, fmt.Errorf("remove existing socket: %w", err)
	}

	if err := os.MkdirAll(filepath.Dir(socket), 0o755); err != nil {
		return nil, fmt.Errorf("create socket directory: %w", err)
	}

	listener, err := net.Listen("unix", socket)
	if err != nil {
		return nil, fmt.Errorf("listen on socket: %w", err)
	}

	if err := os.Chmod(socket, 0o660); err != nil {
		listener.Close()
		return nil, fmt.Errorf("set socket permissions: %w", err)
	}

	return listener, nil
}
