// internal/operations/delete.go

package operations

import "fmt"

type DeleteOpts struct {
	ID string
}

func Delete(opts *DeleteOpts) error {
	fmt.Println(opts)

	return nil
}
