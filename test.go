package main

import (
	"encoding/json"

	"github.com/baderkha/library/pkg/store/client/typesense"
	"github.com/baderkha/library/pkg/store/entity"
	"github.com/davecgh/go-spew/spew"
)

func main() {

	documenter := typesense.NewDocumentClient[entity.AccountPublic]("vt0CplF5ePNwcRpPM3lfNbYi1T4qq5oo", "https://t321gvz6cw9xsmfkp-1.a1.typesense.net", true)
	documenter = documenter.WithoutDocAutoAlias()
	acc := &entity.AccountPublic{
		Base: entity.Base{
			ID: "some_id",
		},
		Email: "some@email.com",
	}
	acc.New()
	err := documenter.Index(acc)
	doc, _ := documenter.GetById("44958c70-a836-4700-8b6e-1e392f01fd89")
	//migrator := typesense.NewModelMigration[entity.AccountPublic]("vt0CplF5ePNwcRpPM3lfNbYi1T4qq5oo", "https://t321gvz6cw9xsmfkp-1.a1.typesense.net", false)
	spew.Dump(err)
	res, _ := json.Marshal(doc)
	spew.Dump(string(res))

}
