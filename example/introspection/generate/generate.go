package main

import (
	"log"

	"github.com/99designs/gqlgen/api"
	"github.com/99designs/gqlgen/codegen/config"
	"github.com/99designs/gqlgen/plugin/introauth"
)

func main() {
	cfg, err := config.LoadConfigFromDefaultLocations()
	if nil != err {
		log.Fatal(err)
	}
	err = api.Generate(cfg, api.AddPlugin(introauth.New(introauth.Config{
		Directives: config.StringList{
			"hide",
			"requireAuth",
			"requireOwner",
		},
	})))
	if nil != err {
		log.Fatal(err)
	}
}
