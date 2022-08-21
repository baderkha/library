package typesense

import (
	"github.com/spf13/afero"
)

var (
	fs = afero.NewOsFs()
)

// OverrideFS : change the file ssytem that will be used in the client
//
// call this before constructing the Client . if you want a different file system
//
// see :
//
// https://github.com/spf13/afero#available-backends
//
// // forexample you can have a s3 bucket to store your jsonl files
//
// https://github.com/fclairamb/afero-s3  // (cool right :))
//
//
//
func OverrideFS(newFs afero.Fs) {
	fs = newFs
}

// Client : General Client that contains all operations supported by typesense
//
//
type Client[T any] struct {
	Migration[T]
	DocumentClient[T]
	SearchClient[T]
}
