// internal/operations/create.go

package operations

import "fmt"

type CreateOpts struct {
	ID     string
	Bundle string
}

func Create(opts *CreateOpts) error {
	fmt.Println(opts)

	return nil
}
