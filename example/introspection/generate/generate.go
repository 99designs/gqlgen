package main

import (
	"github.com/99designs/gqlgen/api"
	"github.com/99designs/gqlgen/codegen/config"
	"github.com/99designs/gqlgen/plugin/introspection"
	"log"
)

func main() {
	cfg, err := config.LoadConfigFromDefaultLocations()
	if nil != err {
		log.Fatal(err)
	}
	err = api.Generate(cfg, api.AddPlugin(introspection.New(introspection.Config{
		Directives: config.StringList{
			"hide",
			"requireAuth",
		},
	})))
	if nil != err {
		log.Fatal(err)
	}
}
