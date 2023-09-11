package termui

import (
	"context"
	"fmt"

	"github.com/gdamore/tcell/v2"
	"github.com/rekram1-node/pgterm/internal/postgres"
	"github.com/rivo/tview"
)

type TermUI struct {
	connections map[string]*postgres.Connection
	app         *tview.Application
	pages       *tview.Pages
	finderFocus tview.Primitive
}

func New(ctx context.Context) (*TermUI, error) {
	app := tview.NewApplication()
	termUI := &TermUI{
		app: app,
	}

	c, err := postgres.New(ctx, "generic", "postgresql://postgres:postgres@localhost:5432/generic?sslmode=disable")
	if err != nil {
		return nil, fmt.Errorf("failed to create postgres connection: %v", err)
	}

	termUI.connections = map[string]*postgres.Connection{
		"generic": c,
	}

	if err := termUI.SetupV2(ctx); err != nil {
		return nil, fmt.Errorf("failed to setup terminal ui: %v", err)
	}

	return termUI, nil
}

func (tui *TermUI) Run() error {
	return tui.app.Run()
}

const (
	finderPage = "*finder*"
)

func (tui *TermUI) SetupV2(ctx context.Context) error {
	flex := tview.NewFlex()
	databases := tview.NewList().ShowSecondaryText(false)
	databases.SetBorder(true).SetTitle("Databases")
	flex.AddItem(databases, 0, 1, true)

	for k, v := range tui.connections {
		databases.AddItem(k, "", 0, func() {
			currentConn := newCurrentDB(v, tui.app, flex, func() {
				tui.app.SetFocus(databases)
				// on close we go back to database list
				flex.AddItem(databases, 0, 1, true)
			})

			flex.AddItem(currentConn.schemas, 0, 1, false)
			currentConn.GetSchemas(ctx, databases)
		})
	}

	tui.pages = tview.NewPages().
		AddPage(finderPage, flex, true, true)
	tui.app.SetRoot(tui.pages, true)

	return nil
}

func (tui *TermUI) Setup(ctx context.Context) error {
	databases := tview.NewList().ShowSecondaryText(false)
	databases.SetBorder(true).SetTitle("Databases")
	columns := tview.NewTable().SetBorders(true)
	columns.SetBorder(true).SetTitle("Columns")
	schemas := tview.NewList()
	schemas.ShowSecondaryText(false).
		SetDoneFunc(func() {
			schemas.Clear()
			columns.Clear()
			tui.app.SetFocus(databases)
		})
	schemas.SetBorder(true).SetTitle("Schemas")
	// Create the layout.
	flex := tview.NewFlex().
		AddItem(databases, 0, 1, true).
		AddItem(schemas, 0, 1, false).
		AddItem(columns, 0, 3, false)

	// could add schema spec here
	schema := "public"

	for k, v := range tui.connections {
		databases.AddItem(k, "", 0, func() {
			columns.Clear()
			schemas.Clear()

			// currentConnection := newCurrentDB(v, tui.app, func() {
			// 	tui.app.SetFocus(databases)
			// })
			// _ = currentConnection
			// schemaList,currentConnection.GetSchemas(ctx)

			schemaList, err := v.GetSchemas(ctx)
			if err != nil {
				panic(err) // fix
			}
			for _, name := range schemaList {
				schemas.AddItem(name, "", 0, nil)
			}

			tui.app.SetFocus(schemas)

			schemas.SetChangedFunc(func(i int, tableName string, t string, s rune) {
				columns.Clear()
				dbColumns, err := v.GetColumns(ctx, schema, tableName)
				if err != nil {
					panic(err)
				}
				columns.SetCell(0, 0, &tview.TableCell{Text: "Name", Align: tview.AlignCenter, Color: tcell.ColorYellow}).
					SetCell(0, 1, &tview.TableCell{Text: "Type", Align: tview.AlignCenter, Color: tcell.ColorYellow}).
					SetCell(0, 2, &tview.TableCell{Text: "Size", Align: tview.AlignCenter, Color: tcell.ColorYellow}).
					SetCell(0, 3, &tview.TableCell{Text: "Null", Align: tview.AlignCenter, Color: tcell.ColorYellow}).
					SetCell(0, 4, &tview.TableCell{Text: "Constraint", Align: tview.AlignCenter, Color: tcell.ColorYellow})

				for _, c := range dbColumns {
					color := tcell.ColorWhite
					if c.ConstraintType.Valid {
						color = map[string]tcell.Color{
							"CHECK":       tcell.ColorGreen,
							"FOREIGN KEY": tcell.ColorDarkMagenta,
							"PRIMARY KEY": tcell.ColorRed,
							"UNIQUE":      tcell.ColorDarkCyan,
						}[c.ConstraintType.String]
					}

					columns.SetCell(c.OrdinalPosition, 0, &tview.TableCell{Text: c.Name, Color: color}).
						SetCell(c.OrdinalPosition, 1, &tview.TableCell{Text: c.DataType, Color: color}).
						SetCell(c.OrdinalPosition, 2, &tview.TableCell{Text: c.SizeText, Align: tview.AlignRight, Color: color}).
						SetCell(c.OrdinalPosition, 3, &tview.TableCell{Text: c.IsNullable, Align: tview.AlignRight, Color: color}).
						SetCell(c.OrdinalPosition, 4, &tview.TableCell{Text: c.ConstraintType.String, Align: tview.AlignLeft, Color: color})
				}
				schemas.SetCurrentItem(0)

				schemas.SetSelectedFunc(func(i int, tableName string, t string, s rune) {
					tui.content(ctx, v.Name(), schema, tableName)
				})
			})
		})
	}

	tui.pages = tview.NewPages().
		AddPage(finderPage, flex, true, true)
	tui.app.SetRoot(tui.pages, true)

	return nil
}

const (
	rowCount = 50
)

func (tui *TermUI) content(ctx context.Context, dbName, schema, table string) {
	tui.finderFocus = tui.app.GetFocus()

	pageName := dbName + "." + schema + "." + table
	if tui.pages.HasPage(pageName) {
		tui.pages.SwitchToPage(pageName)
		return
	}

	// We display the data in a table embedded in a frame.
	tableFrame := tview.NewTable().
		SetFixed(1, 0).
		SetSeparator(tview.BoxDrawingsLightHorizontal).
		SetBordersColor(tcell.ColorYellow)
	frame := tview.NewFrame(tableFrame).
		SetBorders(0, 0, 0, 0, 0, 0)
	frame.SetBorder(true).
		SetTitle(fmt.Sprintf(`Contents of table "%s"`, table))

	// How many rows does this table have?
	var rowCount int
	err := tui.connections[dbName].QueryRow(ctx, fmt.Sprintf("select count(*) from %s", table)).Scan(&rowCount)
	if err != nil {
		panic(err)
	}

	// loadRows := func(offset int) {
	// 	rows, err := tui.connections[dbName].GetRows(ctx, schema, table, rowCount, offset)
	// 	if err != nil {
	// 		panic(err)
	// 	}

	// 	for i, r := range rows {
	// 		row := tableFrame.GetRowCount()
	// 		for k, v := range r {
	// 			column := k
	// 			tableFrame.SetCell(0, i, &tview.TableCell{
	// 				Text:  column,
	// 				Align: tview.AlignCenter,
	// 				Color: tcell.ColorYellow,
	// 			})
	// 			tableFrame.SetCell(row, i, &tview.TableCell{
	// 				Text:  v,
	// 				Align: tview.AlignRight,
	// 				Color: tcell.ColorDarkCyan,
	// 			})
	// 		}
	// 	}
	// 	frame.Clear()
	// 	loadMore := ""
	// 	if tableFrame.GetRowCount()-1 < rowCount {
	// 		loadMore = " - press Enter to load more"
	// 	}
	// 	loadMore = fmt.Sprintf("Loaded %d of %d rows%s", tableFrame.GetRowCount()-1, rowCount, loadMore)

	// 	frame.AddText(loadMore, false, tview.AlignCenter, tcell.ColorYellow)
	// }

	// loadRows(0)

	tableFrame.SetDoneFunc(func(key tcell.Key) {
		switch key {
		case tcell.KeyEscape:
			// Go back to Finder.
			tui.pages.SwitchToPage(finderPage)
			if tui.finderFocus != nil {
				tui.app.SetFocus(tui.finderFocus)
			}
		case tcell.KeyEnter:
			// Load the next batch of rows.
			// loadRows(tableFrame.GetRowCount() - 1)
			tableFrame.ScrollToEnd()
		}
	})

	tui.pages.AddPage(pageName, frame, true, true)
}
