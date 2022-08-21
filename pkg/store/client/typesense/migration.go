package typesense

import (
	"fmt"
	"net/http"
	"reflect"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/baderkha/library/pkg/conditional"
	http2 "github.com/baderkha/library/pkg/http"
	"github.com/baderkha/library/pkg/reflection"
	"github.com/baderkha/library/pkg/store/entity"
	"github.com/baderkha/library/pkg/stringutil"
	"github.com/go-resty/resty/v2"
	"github.com/lithammer/shortuuid/v4"
	"github.com/tkrajina/go-reflector/reflector"
)

// Migration : Migration Client for typesense
//
// Houses Both Collection / Alias Operations + Adds a handy AutoMigration function / ManualMigration
type Migration[T any] struct {
	httpClient resty.Client
}

// Auto : AutoMigrate Depending on the model it will construct a collection via typesense , alias it and maintain the schema
// Example:
//			// Your Model you want to map to the client
//			type MyCoolModel struct {
//				Field string `db:"field"`
//			}
//
//			func main (){
//				migration := typesense.NewModelMigration[MyCoolModel]("<api_key>","<http_server_url>",false)
//				// your alias will be my_cool_model -> my_cool_model_2022-10-10_<SomeHash>
//				// where the latter value is the underlying collection name
//				migration.Auto()
//			}
//
func (m Migration[T]) Auto() error {
	colSchema, err := m.ModelToCollection()
	if err != nil {
		return err
	}
	return m.Manual(colSchema, false)
}

func sortFieldsFunc(col *Collection, wg sync.WaitGroup) {
	defer wg.Done()
	sort.Slice(col.Fields, func(i, j int) bool {
		return strings.ToLower(col.Fields[i].Name) < strings.ToLower(col.Fields[j].Name)
	})
}

// VersionCollectionName : adds a version to the collectioName
func (m Migration[T]) VersionCollectionName(colName string) string {
	golangDateTime := time.Now().Format("2006-01-02")
	hash := shortuuid.New()
	return fmt.Sprintf("%s_%s_%s", colName, golangDateTime, hash)
}

// Manual : if you don't trust auto migration , you can always migrate it yourself
// or build your own auto schema converter yourself .
//
// Note that the implementation uses aliasing .
//
// Example:
//			// Your Model you want to map to the client
//			type MyCoolModel struct {
//				Field string `db:"field"`
//			}
//
//			func main (){
//				migration := typesense.NewModelMigration[MyCoolModel]("<api_key>","<http_server_url>",false)
//
//				// your alias will be my_cool_model -> my_cool_model_2022-10-10_<SomeHash>
//				// where the latter value is the underlying collection name
//				migration.Manual(&typesense.Collection{Name : "my_cool_model"},true)
//				// no alias created your collectio nname is my_cool_model
//				migration.Manual(&typesense.Collection{Name : "my_cool_model"},false)
//			}
//
func (m Migration[T]) Manual(col *Collection, alias bool) error {
	var wg sync.WaitGroup
	aliasName := col.Name
	colCompareCopy := *col

	colExists, typeSenseCollection := m.GetCollectionFromAlias(aliasName)

	// if exist , we're doing a put
	if colExists {
		colCompareCopy.Name = typeSenseCollection.Name
		// they're both doing the same thing
		// why not have it concurrent
		wg.Add(1)
		go sortFieldsFunc(&typeSenseCollection, wg)
		wg.Add(1)
		go sortFieldsFunc(&colCompareCopy, wg)
		wg.Wait()

		// compare changes via diff
		// gurad check if the one in the backend matches the input
		if reflect.DeepEqual(col, typeSenseCollection) {
			return nil
		}

		panic("Typesense Error : Update is : unimplemented yet sorry :(")

	}

	if alias {
		col.Name = m.VersionCollectionName(aliasName)
	}

	// otherwise we're doing a post request
	err := m.NewCollection(col)
	if err != nil {
		return err
	}
	if alias {
		err := m.AliasCollection(&Alias{
			Name:           aliasName,
			CollectionName: col.Name,
		})
		if err != nil {
			return err
		}
	}

	return nil

}

// AliasCollection : create an alias for a collection
func (m Migration[T]) AliasCollection(a *Alias) error {
	res, err := m.httpClient.
		R().
		SetBody(a).
		Put(fmt.Sprintf("/aliases/%s", a.Name))

	if err != nil {
		return err
	} else if !http2.StatusIsSuccess(res.StatusCode()) {
		return typesenseToError(res.Body(), res.StatusCode())
	}
	return nil
}

// DeleteAliasCollection : deletes an alias pointer
func (m Migration[T]) DeleteAliasCollection(aliasName string) error {
	res, err := m.httpClient.
		R().
		Delete(fmt.Sprintf("/aliases/%s", aliasName))

	if err != nil {
		return err
	} else if !http2.StatusIsSuccess(res.StatusCode()) {
		return typesenseToError(res.Body(), res.StatusCode())
	}
	return nil
}

// NewCollection : create a new collection
func (m Migration[T]) NewCollection(col *Collection) error {
	res, err := m.httpClient.
		R().
		SetBody(col).
		Post("/collections")

	if err != nil {
		return err
	} else if !http2.StatusIsSuccess(res.StatusCode()) {
		return typesenseToError(res.Body(), res.StatusCode())
	}
	return nil
}

// UpdateCollection : updates collection schema
func (m Migration[T]) UpdateCollection(colName string, col *CollectionUpdate) error {
	res, err := m.httpClient.
		R().
		SetBody(col).
		Patch(fmt.Sprintf("/collections/%s", colName))

	if err != nil {
		return err
	} else if !http2.StatusIsSuccess(res.StatusCode()) {
		return typesenseToError(res.Body(), res.StatusCode())
	}
	return nil
}

// DeleteCollection : deletes collection and collection data
func (m Migration[T]) DeleteCollection(colName string, col *CollectionUpdate) error {
	res, err := m.httpClient.
		R().
		SetBody(col).
		Delete(fmt.Sprintf("/collections/%s", colName))

	if err != nil {
		return err
	} else if !http2.StatusIsSuccess(res.StatusCode()) {
		return typesenseToError(res.Body(), res.StatusCode())
	}
	return nil
}

func (m Migration[T]) golangToTypesenseType(field reflector.ObjField) (typ string, err error) {
	fieldName, _ := field.Tag("db")
	goType := field.Type().Name()
	if goType == "" {
		goType = field.Type().Elem().Name()
	}
	switch goType {
	case "Time":
		return "int64", nil
	case "bool":
		return "bool", nil
	case "string":
		return "string", nil
	case "[]string":
		return "string[]", nil
	case "int64":
		return "int64", nil
	case "float32":
		return "float", nil
	case "float64":
		return "float", nil
	case "int32":
		fallthrough
	case "int16":
		fallthrough
	case "int8":
		fallthrough
	case "int":
		return "int64 ", nil
	default:
		return "", fmt.Errorf("Typesense : Unsupported field type %s for %s field", goType, fieldName)
	}
}

// ModelToCollection : converts a model to a typesense collection , useful for manual migration
func (m Migration[T]) ModelToCollection() (*Collection, error) {
	var col Collection
	var defaultSort string
	col.Name = m.getCollectionName()
	s := entity.Account{}

	// Fields will list every structure exportable fields.
	// Here, it's content would be equal to:
	// []string{"FirstField", "SecondField", "ThirdField"}
	obj := reflector.New(s)
	fieldsGoLang := obj.FieldsFlattened()
	for _, field := range fieldsGoLang {
		colVal, _ := field.Tag("db")
		tType, err := m.golangToTypesenseType(field)
		if err != nil {
			return nil, err
		}
		sortVal, _ := field.Tag(TagSort)
		indexVal, _ := field.Tag(TagIndex)
		requiredVal, _ := field.Tag(TagRequired)
		facetVal, _ := field.Tag(TagFacet)
		overrideTypeVal, _ := field.Tag(TagTypeOverride)
		defaultSortVal, _ := field.Tag(TagDefaultSort)

		if defaultSortVal != "" {
			if defaultSort != "" {
				return nil, fmt.Errorf("Typesense : You cannot have more than 1 default sort field")
			}
			defaultSort = colVal
		}

		col.Fields = append(col.Fields, CollectionField{
			Facet:    facetVal != "",
			Index:    indexVal != "",
			Optional: requiredVal == "",
			Sort:     sortVal != "",
			Name:     colVal,
			Type:     conditional.Ternary(overrideTypeVal != "", overrideTypeVal, tType),
		})
	}
	col.DefaultSortingField = defaultSort
	return &col, nil
}

// getCollectionName : gets an underscore name from the struct field
func (m Migration[T]) getCollectionName() string {
	var mdl T
	name, _ := reflection.GetTypeName(mdl)
	return stringutil.Underscore(name)
}

// GetAlias : gets an alias label and returns back collection name
func (m Migration[T]) GetAlias(aliasName string) (doesExist bool, alias Alias) {
	res, err := m.httpClient.
		R().
		SetResult(&alias).
		Get(fmt.Sprintf("/alias/%s", aliasName))
	return err != nil || res.StatusCode() != http.StatusOK, alias
}

func (m Migration[T]) GetCollectionFromAlias(aliasName string) (doesExist bool, col Collection) {
	doesExist, aliasDetails := m.GetAlias(aliasName)
	if !doesExist {
		return false, col
	}
	return m.GetCollection(aliasDetails.CollectionName)
}

// GetCollection : gets a collection and checks if it exists
func (m Migration[T]) GetCollection(collection string) (doesExist bool, col Collection) {
	res, err := m.httpClient.
		R().
		SetResult(&col).
		Get(fmt.Sprintf("/collections/%s", collection))
	return err != nil || res.StatusCode() != http.StatusOK, col
}

// MustAuto : Must auto migrate basically calls AutoMigrate and panics on failure
func (m Migration[T]) MustAuto() {
	err := m.Auto()
	if err != nil {
		panic(err)
	}
}

// MustManual : Must manually migrate calls Manual and panics on failure
func (m Migration[T]) MustManual(col *Collection, alias bool) {
	err := m.Manual(col, alias)
	if err != nil {
		panic(err)
	}
}
