package typesense

import (
	"fmt"
	"net/http"

	"github.com/baderkha/library/pkg/reflection"
	"github.com/baderkha/library/pkg/stringutil"
	"github.com/go-resty/resty/v2"
)

// Migration : Migration Client for typesene , can be called independently of the main client
type Migration[T any] struct {
	httpClient          resty.Client
	defualtSortingField string
}

// AutoMigrate : AutoMigrate Depending on the model it will construct a collection via typesense
func (m Migration[T]) AutoMigrate() error {
	colSchema := m.ModelToCollection()
	return m.ManuallyMigrate(colSchema)
}

// ManuallyMigrate : if you don't trust auto migration , you can always migrate it yourself , or build your own auto schema converter yourself
func (m Migration[T]) ManuallyMigrate(col *Collection) error {
	colName := col.Name
	colExists := m.CheckIfCollectionExists(colName)
	// if exist , we're doing a put
	if colExists {
		return nil
	}
	// otherwise we're doing a post request
	return nil
}

// ModelToCollection : converts a model to a collection item
func (m Migration[T]) ModelToCollection() *Collection {
	var col Collection
	col.Name = m.getCollectionName()
	return &col
}

// getCollectionName : gets an underscore name from the struct field
func (m Migration[T]) getCollectionName() string {
	var mdl T
	name, _ := reflection.GetTypeName(mdl)
	return stringutil.Underscore(name)
}

// CheckIfCollectionExists : checks if collection is already there
func (m Migration[T]) CheckIfCollectionExists(collection string) bool {
	res, err := m.httpClient.
		R().
		Get(fmt.Sprintf("/collections/%s", collection))
	return err != nil || res.StatusCode() != http.StatusOK
}

// MustAutoMigrate : Must auto migrate basically calls AutoMigrate and panics on failure
func (m Migration[T]) MustAutoMigrate() {
	err := m.AutoMigrate()
	if err != nil {
		panic(err)
	}
}

// MustManuallyMigrate : Must manually migrate calls ManuallyMigrate and panics on failure
func (m Migration[T]) MustManuallyMigrate(col *Collection) {
	err := m.ManuallyMigrate(col)
	if err != nil {
		panic(err)
	}
}
