// internal/operations/start.go

package operations

import "fmt"

type StartOpts struct {
	ID string
}

func Start(opts *StartOpts) error {
	fmt.Println(opts)

	return nil
}
