package main

import (
	"context"
	"fmt"
	"log"

	"github.com/rekram1-node/pgterm/cmd"
	"github.com/rekram1-node/pgterm/internal/postgres"
	"github.com/rekram1-node/pgterm/internal/termui"
)

func main() {
	ctx := context.Background()
	pgurl := "postgresql://postgres:postgres@localhost:5432/generic?sslmode=disable"

	if true {
		ui, err := termui.New(ctx)
		if err != nil {
			log.Fatal(err)
		}

		if err := ui.Run(); err != nil {
			log.Fatal(err)
		}
		return
	}

	conn, err := postgres.New(ctx, "generic", pgurl)
	if err != nil {
		log.Fatal(err)
	}
	schemas, err := conn.GetSchemas(ctx)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println(schemas)

	tables, err := conn.GetTables(ctx, "public")
	if err != nil {
		log.Fatal(err)
	}

	fmt.Println(tables)

	columns, err := conn.GetColumns(ctx, "public", "users")
	if err != nil {
		log.Fatal(err)
	}

	for _, c := range columns {
		fmt.Println(c.Name)
	}

	rows, err := conn.GetRows(ctx, "public", "users", 10, 0)
	if err != nil {
		log.Fatal(err)
	}

	for i, r := range rows {
		fmt.Println("row number:", i)
		for k, v := range r {
			fmt.Printf("%s: %v, ", k, v)
		}
		fmt.Println()
	}

	return
	cmd.Execute()
}
