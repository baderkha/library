package json

import (
	"encoding/json"
	"os"
)

// MustReadJSONFromFile : casts file to a schema specified
func MustReadJSONFromFile[T any](filePath string) T {
	var res T
	data, err := os.ReadFile(filePath)
	if err != nil {
		panic(err)
	}
	err = json.Unmarshal(data, &res)
	if err != nil {
		panic(err)
	}
	return res
}
