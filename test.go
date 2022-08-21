package main

import (
	"github.com/baderkha/library/pkg/store/client/typesense"
	"github.com/baderkha/library/pkg/store/entity"
	"github.com/davecgh/go-spew/spew"
)

func main() {

	// Fields will list every structure exportable fields.
	// Here, it's content would be equal to:
	// []string{"FirstField", "SecondField", "ThirdField"}
	migrator := typesense.Migration[entity.Account]{}
	spew.Dump(migrator.ModelToCollection())

}
