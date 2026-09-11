// Command seed creates an employee account directly in the database.
// Needed to bootstrap the first admin, since accounts are otherwise only
// creatable by an existing admin/manager via the API. Also handy for
// seeding demo/test accounts of any role or pay type.
//
// Usage: go run ./cmd/seed <db_path> <employee_code> <name> <pin> [role] [pay_type] [pay_rate] [is_temp]
//
//	role:     admin | manager | staff   (default: admin)
//	pay_type: monthly | hourly          (default: monthly)
//	pay_rate: number                    (default: 0)
//	is_temp:  true | false              (default: false)
package main

import (
	"fmt"
	"os"
	"strconv"

	"attendance-mgmt/backend/internal/auth"
	"attendance-mgmt/backend/internal/db"
)

func main() {
	if len(os.Args) < 5 {
		fmt.Fprintln(os.Stderr, "usage: seed <db_path> <employee_code> <name> <pin> [role] [pay_type] [pay_rate] [is_temp]")
		os.Exit(1)
	}
	dbPath, code, name, pin := os.Args[1], os.Args[2], os.Args[3], os.Args[4]

	role := argOr(5, "admin")
	payType := argOr(6, "monthly")
	payRate, err := strconv.ParseFloat(argOr(7, "0"), 64)
	if err != nil {
		fmt.Fprintln(os.Stderr, "invalid pay_rate:", err)
		os.Exit(1)
	}
	isTemp, err := strconv.ParseBool(argOr(8, "false"))
	if err != nil {
		fmt.Fprintln(os.Stderr, "invalid is_temp:", err)
		os.Exit(1)
	}

	conn, err := db.Open(dbPath)
	if err != nil {
		fmt.Fprintln(os.Stderr, "open db:", err)
		os.Exit(1)
	}
	defer conn.Close()

	hash, err := auth.HashPIN(pin)
	if err != nil {
		fmt.Fprintln(os.Stderr, "hash pin:", err)
		os.Exit(1)
	}

	_, err = conn.Exec(`
		INSERT INTO employees (employee_code, name, role, pin_hash, pay_type, pay_rate, is_temp)
		VALUES (?, ?, ?, ?, ?, ?, ?)
	`, code, name, role, hash, payType, payRate, isTemp)
	if err != nil {
		fmt.Fprintln(os.Stderr, "insert employee:", err)
		os.Exit(1)
	}

	fmt.Printf("created %s %s (%s)\n", role, name, code)
}

func argOr(i int, fallback string) string {
	if i < len(os.Args) {
		return os.Args[i]
	}
	return fallback
}
