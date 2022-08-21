package typesense

// Client : I hate the way the unofficial typesense client is so i'm going to make my own using generics
//			Generics Based Typesense Rest Api Wrapper Client
type Client[T any] struct {
	Migration[T] // a client composes migration methods
}
