// Package typense : This package contains a slew of clients that implement the typesense restful interface
//
// Each feature Section under the documentation is abstracted to its own client
//
// IE There is a :
//
// - Migration Client => manages collections / aliases
//
// - Search Client    => Allows advanced search
//
// - Document Client  => Allows indexing (inserting / del / update) Documents
//
// - Cluster Client   => Manages cluster / gets health and other metrics
//
// - Main Client      => A facade for all the clients the fat client that has everything if you're lazy like me
//
// Additionally there are an interfaces for each client as well as a `mock` implementations of the interfaces if you need
// it in a test setting  (built using testify mock package) . However , You are responsible for breaking changes in your testing setup.
//
// Logging is also supported (it will log the outgoing http requests and http responses from typesense)
//
// Final Note :
//
// Create/Update/Delete Operations do not return anything except for errors if there are any . This was a concious design decision .
// Given that Typesense Returns correct status codes ie no need to read the json body data.
// If you disagree feel free to add your implementation and expand the interface
//
//
// See : https://github.com/baderkha
//
package typesense

import (
	"fmt"

	"github.com/go-resty/resty/v2"
)

const (
	typesenseErrPrefix = "TypeSenseClient : Bad Response : With Code"
)

const (
	// TagSort : attach this to your struct field tsense_sort
	//
	// Example :
	//           // your model
	//			 type Model struct {
	//				Field string `tsense_sort:"1"` // this will tell typesense you want this field sorted
	//			 }
	//
	TagSort = "tsense_sort"
	// TagIndex : attach this to your struct field tsense_index
	//
	// Example :
	//           // your model
	//			 type Model struct {
	//				Field string `tsense_index:"1"` // this will tell typesense you want this field indexed
	//			 }
	//
	TagIndex = "tsense_index"
	// TagRequired : attach this to your struct field tsense_required
	//
	// Example :
	//           // your model
	//			 type Model struct {
	//				Field string `tsense_required:"1"` // this will tell typesense you want this field to be required during creates
	//			 }
	//
	TagRequired = "tsense_required"
	// TagRequired : attach this to your struct field tsense_facet
	//
	// Example :
	//           // your model
	//			 type Model struct {
	//				Field string `tsense_facet:"1"` // this will tell typesense you want this field as a facet
	//			 }
	//
	TagFacet = "tsense_facet"
	// TagTypeOverride : attach this to your struct field tsense_type
	//
	// Example :
	//           // your model
	//			type Model struct {
	//				Field int8 `tsense_type:"int32"` // this will tell typesense you want
	//												 // this field to override the type instead of the auto type (int64)
	//			}
	//
	TagTypeOverride = "tsense_type"
	// TagTypeOverride : attach this to your struct field tsense_default_sort
	//
	// Example :
	//           // your model
	//			type Model struct {
	//				Field string `tsense_default_sort:"1"` // this will tell typesense you want
	//												   // this field to be the default sort field
	//			}
	//
	TagDefaultSort = "tsense_default_sort"
)

func typesenseToError(responseBody []byte, statusCode int) error {
	return fmt.Errorf("%s : %d  : %s", typesenseErrPrefix, statusCode, string(responseBody))
}

// CollectionField : field for a typesense collection
type CollectionField struct {
	Facet    bool `json:"facet"`
	Index    bool `json:"index"`
	Optional bool `json:"optional"`
	Sort     bool `json:"sort"`

	Name string `json:"name"`
	Type string `json:"type"`
}

// Collection : typesense collection
type Collection struct {
	Name                string            `json:"name"`
	Fields              []CollectionField `json:"fields"`
	DefaultSortingField string            `json:"default_sorting_field"`
}

// CollectionField : field for a typesense collection
type CollectionFieldUpdate struct {
	Facet    bool   `json:"facet"`
	Index    bool   `json:"index"`
	Optional bool   `json:"optional"`
	Sort     bool   `json:"sort"`
	Drop     bool   `json:"drop"`
	Name     string `json:"name"`
	Type     string `json:"type"`
}

// Collection : typesense collection
type CollectionUpdate struct {
	Fields              []CollectionField `json:"fields"`
	DefaultSortingField string            `json:"default_sorting_field"`
}

// Alias : alias to a collection
type Alias struct {
	Name           string `json:"name "`
	CollectionName string `json:"collection_name"`
}

func newHTTPClient(apiKey, host string, logging bool) *resty.Client {
	return resty.
		New().
		SetHeaders(map[string]string{
			"Content-Type":        "application/json",
			"X-TYPESENSE-API-KEY": apiKey,
		}).
		SetBaseURL(host).
		SetDebug(logging)
}

// NewModelMigration : Migration if you want to use Model Dependent migration ie tie your migration client to a
//A Specific struct declaration
func NewModelMigration[T any](apiKey string, host string, logging bool) *Migration[T] {
	return &Migration[T]{
		httpClient: *newHTTPClient(apiKey, host, logging),
	}
}

// NewManualMigration : Migration if you want to do your own thing and use the low level wrapper methods for the rest calls
//
// Disclaimer :
// 				// Do not use Auto Method
// 				migrator := typesense.NewManualMigration("","","")
//				migrator.Auto() // should panic or error or give you a bad status code
//
//				// instead you will have to make the calls yourself
func NewManualMigration(apiKey string, host string, logging bool) *Migration[any] {
	return &Migration[any]{
		httpClient: *newHTTPClient(apiKey, host, logging),
	}
}
