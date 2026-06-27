//go:build ignore

package main

import (
	"context"
	"fmt"
	"github.com/jackc/pgx/v5/pgxpool"
	"os"
)

func main() {
	pool, err := pgxpool.New(context.Background(), "postgres://postgres:postgres@127.0.0.1:5432/app_db?sslmode=disable")
	if err != nil {
		fmt.Println("connect:", err)
		os.Exit(1)
	}
	defer pool.Close()

	rows, err := pool.Query(context.Background(),
		`SELECT id, action, entity_type, entity_id, user_id, metadata, created_at FROM audit_logs WHERE organization_id = $1 ORDER BY created_at DESC LIMIT 5 OFFSET 0`,
		"3db43950-8945-439e-a03f-61722b8d3c20")
	if err != nil {
		fmt.Println("query error:", err)
		os.Exit(1)
	}
	defer rows.Close()

	for rows.Next() {
		var id, action, entityType, createdAt string
		var entityID, userID *string
		var metadata []byte
		err := rows.Scan(&id, &action, &entityType, &entityID, &userID, &metadata, &createdAt)
		if err != nil {
			fmt.Println("scan error:", err)
			os.Exit(1)
		}
		fmt.Printf("OK: %s %s %s\n", id, action, createdAt)
	}
	if err := rows.Err(); err != nil {
		fmt.Println("rows error:", err)
		os.Exit(1)
	}
	fmt.Println("SUCCESS")
}
