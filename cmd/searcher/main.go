// Copyright 2025 Poiesic Systems
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.


package main

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"strings"

	"github.com/poiesic/memorit"
	"github.com/poiesic/memorit/core"
)

func init() {
	handler := slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelInfo,
	})
	slog.SetDefault(slog.New(handler))
}

func main() {
	db, err := memorit.NewDatabase("./history_db")
	if err != nil {
		panic(err)
	}
	defer db.Close()
	searcher, err := db.NewSearcher()
	if err != nil {
		panic(err)
	}

	ctx := context.Background()
	var results []*core.SearchResult
	if len(os.Args) > 1 {
		results, err = searcher.FindSimilar(ctx, strings.Join(os.Args[1:], " "), 5)
	} else {
		results, err = searcher.FindSimilar(ctx, "lantern", 5)
	}
	if err != nil {
		panic(err)
	}

	fmt.Printf("Found %d hits\n", len(results))
	for i, hit := range results {
		fmt.Printf("%d: '%s' (%d)[%0.3f]\n", i, hit.Record.Contents, hit.Record.Id, hit.Score)
	}
}
