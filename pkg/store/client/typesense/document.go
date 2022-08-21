package typesense

import "github.com/go-resty/resty/v2"

// DocumentClient : Document client (meant for simple gets , post , patch , deletes
type DocumentClient[T any] struct {
	httpClient resty.Client
}
