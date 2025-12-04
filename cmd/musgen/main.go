package main

import (
	"os"
	"reflect"
	"strings"

	musgen "github.com/mus-format/musgen-go/mus"
	genops "github.com/mus-format/musgen-go/options/generate"
	structops "github.com/mus-format/musgen-go/options/struct"
	typeops "github.com/mus-format/musgen-go/options/type"
	"github.com/poiesic/memorit/core"
)

func main() {
	cwd, err := os.Getwd()
	if err != nil {
		panic(err)
	}
	// If we're in the core subpackage, cd up to project root
	if strings.HasSuffix(cwd, "core") {
		if err := os.Chdir(".."); err != nil {
			panic(err)
		}
	}
	g, err := musgen.NewCodeGenerator(
		genops.WithPkgPath("github.com/poiesic/memorit/core"),
	)
	if err != nil {
		panic(err)
	}

	g.AddDefinedType(reflect.TypeFor[core.SpeakerType]())
	g.AddDefinedType(reflect.TypeFor[core.ID]())

	// Unix milli timestamps
	opts := typeops.WithTimeUnit(typeops.Micro)
	err = g.AddStruct(reflect.TypeFor[core.ChatRecord](),
		structops.WithField(),
		structops.WithField(),
		structops.WithField(),
		structops.WithField(opts),
		structops.WithField(opts),
		structops.WithField(opts),
		structops.WithField(),
		structops.WithField(),
		structops.WithField())
	if err != nil {
		panic(err)
	}

	err = g.AddStruct(reflect.TypeFor[core.Concept](),
		structops.WithField(),
		structops.WithField(),
		structops.WithField(),
		structops.WithField(),
		structops.WithField(opts),
		structops.WithField(opts))
	if err != nil {
		panic(err)
	}

	err = g.AddStruct(reflect.TypeFor[core.ConceptRef](),
		structops.WithField(),
		structops.WithField())
	if err != nil {
		panic(err)
	}

	err = g.AddStruct(reflect.TypeFor[core.Checkpoint](),
		structops.WithField(),
		structops.WithField(),
		structops.WithField(opts))
	if err != nil {
		panic(err)
	}

	bs, err := g.Generate()
	if err != nil {
		panic(err)
	}

	err = os.WriteFile("./core/records_mus.gen.go", bs, 0644)
	if err != nil {
		panic(err)
	}
}
