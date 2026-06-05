//go:build ignore
// +build ignore

// Package main 是 ent 代码生成的入口。
//
// 运行：go generate ./ent
package main

import (
	"log"

	"entgo.io/ent/entc"
	"entgo.io/ent/entc/gen"
)

func main() {
	if err := entc.Generate("./schema", &gen.Config{}); err != nil {
		log.Fatalf("running ent codegen: %v", err)
	}
}
