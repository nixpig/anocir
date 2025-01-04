// internal/operations/state.go

package operations

import "fmt"

type StateOpts struct {
	ID string
}

func State(opts *StateOpts) (string, error) {
	fmt.Println(opts)

	return "", nil
}
