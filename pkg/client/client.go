package client

import (
	api "github.com/elixirhealth/directory/pkg/directoryapi"
	"google.golang.org/grpc"
)

// NewInsecure returns a new DirectoryClient without any TLS on the connection.
func NewInsecure(address string) (api.DirectoryClient, error) {
	cc, err := grpc.Dial(address, grpc.WithInsecure())
	if err != nil {
		return nil, err
	}
	return api.NewDirectoryClient(cc), nil
}
