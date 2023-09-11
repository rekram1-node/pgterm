package postgres

import (
	"context"
	"encoding/hex"
	"fmt"
	"strconv"
	"strings"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
)

// Interface for Postgres Connection
type DBReader interface {
	GetTables(ctx context.Context, schema string) ([]string, error)
	GetColumns(ctx context.Context, schema, table string) ([]Column, error)
	GetRows(ctx context.Context, schema, table string, rowCount, offset int) ([]map[string]string, error)
}

type Connection struct {
	name string
	*pgx.Conn
}

func New(ctx context.Context, name, pgurl string) (*Connection, error) {
	db, err := pgx.Connect(ctx, pgurl)
	if err != nil {
		return nil, err
	}

	return &Connection{
		name,
		db,
	}, nil
}

func (c *Connection) Name() string {
	return c.name
}

func (c *Connection) GetSchemas(ctx context.Context) ([]string, error) {
	query := `
	SELECT SCHEMA_NAME
	FROM INFORMATION_SCHEMA.SCHEMATA
	WHERE SCHEMA_NAME NOT IN ('information_schema', 'pg_catalog', 'pg_toast')`
	rows, err := c.Query(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("failed to find schemas: %v", err)
	}
	schemas, err := pgx.CollectRows(rows, pgx.RowTo[string])
	if err != nil {
		return nil, fmt.Errorf("failed to convert rows to string slice: %v", err)
	}

	return schemas, nil
}

func (c *Connection) GetTables(ctx context.Context, schema string) ([]string, error) {
	query := fmt.Sprintf(`
	SELECT table_name FROM information_schema.tables 
	WHERE table_schema = '%s'`, schema)
	rows, err := c.Query(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("failed to find tables: %v", err)
	}
	defer rows.Close()

	tables, err := pgx.CollectRows(rows, pgx.RowTo[string])
	if err != nil {
		return nil, fmt.Errorf("failed to convert rows to string slice: %v", err)
	}

	return tables, nil
}

type Column struct {
	Name, IsNullable, DataType, SizeText string
	ConstraintType                       pgtype.Text
	OrdinalPosition                      int
}

func (c *Connection) GetColumns(ctx context.Context, schema, table string) ([]Column, error) {
	query := `
	SELECT c.column_name,
		c.is_nullable,
		c.data_type,
		c.character_maximum_length,
		c.numeric_precision,
		c.numeric_scale,
		c.ordinal_position,
		tc.constraint_type pkey
	FROM   information_schema.columns c
		LEFT JOIN information_schema.constraint_column_usage AS ccu
				ON c.table_schema = ccu.table_schema
					AND c.table_name = ccu.table_name
					AND c.column_name = ccu.column_name
		LEFT JOIN information_schema.table_constraints AS tc
				ON ccu.constraint_schema = tc.constraint_schema
					AND ccu.constraint_name = tc.constraint_name
	WHERE  c.table_schema = $1
		AND c.table_name = $2`

	rows, err := c.Query(ctx, query, schema, table)
	if err != nil {
		return nil, fmt.Errorf("failed to find columns for table [%s]: %v", table, err)
	}
	defer rows.Close()

	var columns []Column
	for rows.Next() {
		var (
			column Column

			size, numericPrecision, numericScale pgtype.Int8
		)
		if err := rows.Scan(
			&column.Name,
			&column.IsNullable,
			&column.DataType,
			&size,
			&numericPrecision,
			&numericScale,
			&column.OrdinalPosition,
			&column.ConstraintType,
		); err != nil {
			return nil, fmt.Errorf("failed to scan row for table [%s]: %v", table, err)
		}

		if size.Valid {
			column.SizeText = strconv.Itoa(int(size.Int64))
		} else if numericPrecision.Valid {
			column.SizeText = strconv.Itoa(int(numericPrecision.Int64))
			if numericScale.Valid {
				column.SizeText += "," + strconv.Itoa(int(numericScale.Int64))
			}
		}

		columns = append(columns, column)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	return columns, nil
}

func (c *Connection) GetRows(ctx context.Context, schema, table string, rowCount, offset int) ([]map[string]string, error) {
	tableSchema := fmt.Sprintf("%s.%s", schema, table)
	query := fmt.Sprintf("SELECT * FROM %s LIMIT $1 OFFSET $2", tableSchema)
	rows, err := c.Query(ctx, query, rowCount, offset)
	if err != nil {
		return nil, fmt.Errorf("failed to find rows for table [%s]: %v", table, err)
	}
	defer rows.Close()
	var tableRows []map[string]string

	for rows.Next() {
		values, err := rows.Values()
		if err != nil {
			return nil, fmt.Errorf("failed to retrieve row values: %v", err)
		}

		rowMap := make(map[string]string)
		for i, val := range values {
			colName := rows.FieldDescriptions()[i].Name
			switch v := val.(type) {
			case [16]uint8:
				rowMap[string(colName)] = formatUUID(v)
			default:
				rowMap[string(colName)] = fmt.Sprint(val)
			}
		}

		tableRows = append(tableRows, rowMap)
	}

	if rows.Err() != nil {
		return nil, err
	}

	return tableRows, nil
}

func formatUUID(uuidBytes [16]byte) string {
	hexStr := hex.EncodeToString(uuidBytes[:])
	if len(hexStr) != 32 {
		return hexStr // Return the hex as-is if it doesn't look like a UUID
	}
	return strings.Join([]string{
		hexStr[0:8],
		hexStr[8:12],
		hexStr[12:16],
		hexStr[16:20],
		hexStr[20:32],
	}, "-")
}
