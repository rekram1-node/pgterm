package termui

import (
	"context"

	"github.com/rekram1-node/pgterm/internal/postgres"
	"github.com/rivo/tview"
)

type currentDatabase struct {
	app             *tview.Application
	flex            *tview.Flex
	conn            *postgres.Connection
	schemas, tables *tview.List
	Schema, Table   string
}

func newCurrentDB(conn *postgres.Connection, app *tview.Application, flex *tview.Flex, doneFunc func()) *currentDatabase {
	schemas := tview.NewList()
	schemas.
		ShowSecondaryText(false).
		SetBorder(true).
		SetTitle("Schemas")
	tables := tview.NewList()
	tables.
		ShowSecondaryText(false).
		SetBorder(true).
		SetTitle("Tables")
	return &currentDatabase{
		app:  app,
		flex: flex,
		conn: conn,
		schemas: schemas.
			SetDoneFunc(func() {
				schemas.Clear()
				flex.RemoveItem(schemas)
				doneFunc()
			}),
		tables: tables.SetDoneFunc(func() {
			tables.Clear()
			flex.RemoveItem(tables)
			flex.AddItem(schemas, 0, 1, true)
			app.SetFocus(schemas)
		}),
	}
}

func (x *currentDatabase) GetSchemas(ctx context.Context, parentList *tview.List) {
	schemas, err := x.conn.GetSchemas(ctx)
	if err != nil {
		panic(err)
	}

	for _, schema := range schemas {
		x.schemas.AddItem(schema, "", 0, nil)
	}

	x.app.SetFocus(x.schemas)
	// we focus on schema list instead of database list
	x.flex.RemoveItem(parentList)

	// x.schemas.
	// x.GetTables(ctx)
	x.schemas.SetSelectedFunc(func(i int, s1, s2 string, r rune) {
		x.flex.AddItem(x.tables, 0, 1, false)
		x.GetTables(ctx, s1)
	})
}

func (x *currentDatabase) GetRows(ctx context.Context, numRows, offset int) ([]map[string]string, error) {
	return x.conn.GetRows(ctx, x.Schema, x.Table, numRows, offset)
}

func (x *currentDatabase) GetColumns(ctx context.Context) ([]postgres.Column, error) {
	return x.conn.GetColumns(ctx, x.Schema, x.Table)
}

func (x *currentDatabase) GetTables(ctx context.Context, schema string) {
	tables, err := x.conn.GetTables(ctx, schema)
	if err != nil {
		panic(err)
	}

	for _, table := range tables {
		x.tables.AddItem(table, "", 0, nil)
	}

	x.app.SetFocus(x.tables)
	// focus on tables rather than schemas
	x.flex.RemoveItem(x.schemas)
}
