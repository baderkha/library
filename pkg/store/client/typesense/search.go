package typesense

import "github.com/go-resty/resty/v2"

type SearchClient[T any] struct {
	httpClient resty.Client
}
