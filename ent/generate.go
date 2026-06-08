//go:build ignore

package main

import (
	"log"

	"entgo.io/ent/entc"
	"entgo.io/ent/entc/gen"
	"entgo.io/ent/schema/field"
)

func main() {
	err := entc.Generate("./schema", &gen.Config{
		Target:   "./",
		Package:  "github.com/stifer/agent-hub/ent",
		IDType:   &field.TypeInfo{Type: field.TypeInt64},
		Features: []gen.Feature{gen.FeatureUpsert, gen.FeatureIntercept},
	})
	if err != nil {
		log.Fatalf("running ent codegen: %v", err)
	}
}
