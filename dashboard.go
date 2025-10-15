// dashboard.go
package main

import (
	"database/sql"
	"fmt"
	"html/template"
	"log"
	"net/http"
	"strconv"
)

// A list of tables we allow to be viewed.
// IMPORTANT: This acts as a whitelist to prevent SQL injection on table names.
var allowedTables = []string{"users", "active_games", "games", "transactions", "daily_rewards"}

// Global variable to hold our parsed templates
var templates = template.Must(template.ParseFiles("templates/index.html", "templates/table.html"))

// dashboardData holds the data passed to the table.html template
type dashboardData struct {
	TableName string
	Columns   []string
	Rows      []map[string]interface{}
}

// StartDashboard initializes and starts the web server in a separate goroutine.
func StartDashboard(db *sql.DB, port string) {
	mux := http.NewServeMux()

	// Create handlers that have access to the database connection pool
	indexHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		handleIndex(w, r)
	})
	tableHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		handleViewTable(w, r, db)
	})

	mux.Handle("/", indexHandler)
	mux.Handle("/table", tableHandler)

	log.Printf("Dashboard starting on http://localhost:%s", port)

	// Run the server in a goroutine so it doesn't block the main thread (our Discord bot).
	go func() {
		if err := http.ListenAndServe(":"+port, mux); err != nil {
			log.Fatalf("Dashboard server failed: %v", err)
		}
	}()
}

// handleIndex serves the home page of the dashboard.
func handleIndex(w http.ResponseWriter, r *http.Request) {
	err := templates.ExecuteTemplate(w, "index.html", allowedTables)
	if err != nil {
		http.Error(w, "Could not render template", http.StatusInternalServerError)
	}
}

// getQueryForTable returns the appropriate SQL query for each table,
// including JOINs to add username where applicable
func getQueryForTable(tableName string) string {
	switch tableName {
	case "users":
		return "SELECT * FROM `users` ORDER BY userid DESC"
	case "active_games":
		return "SELECT * FROM `active_games` ORDER BY userid DESC"
	case "games":
		return `SELECT g.id, g.userid, u.username, g.game_type, g.amount, g.outcome, g.played_at 
				FROM games g 
				LEFT JOIN users u ON g.userid = u.userid 
				ORDER BY g.id DESC`
	case "transactions":
		return "SELECT * FROM `transactions` ORDER BY id DESC"
	case "daily_rewards":
		return `SELECT d.id, d.userid, u.username, d.claim_date, d.streak, d.reward_amount, d.claimed_at 
				FROM daily_rewards d 
				LEFT JOIN users u ON d.userid = u.userid 
				ORDER BY d.id DESC`
	default:
		return fmt.Sprintf("SELECT * FROM `%s` ORDER BY 1 DESC", tableName)
	}
}

// handleViewTable fetches data from a specific table and displays it.
func handleViewTable(w http.ResponseWriter, r *http.Request, db *sql.DB) {
	tableName := r.URL.Query().Get("name")
	limitStr := r.URL.Query().Get("limit")
	offsetStr := r.URL.Query().Get("offset")

	// Parse limit
	limit, err := strconv.Atoi(limitStr)
	if err != nil || limit <= 0 || limit > 1000 {
		limit = 25
	}

	// Parse offset
	offset, err := strconv.Atoi(offsetStr)
	if err != nil || offset < 0 {
		offset = 0
	}

	// Update your query to include OFFSET
	baseQuery := getQueryForTable(tableName) // "SELECT * FROM active_games"
	query := fmt.Sprintf("%s LIMIT ?, ?", baseQuery)
	rows, err := db.Query(query, offset, limit)

	if err != nil {
		http.Error(w, fmt.Sprintf("Database query error: %v", err), http.StatusInternalServerError)
		log.Printf("Error querying table %s: %v", tableName, err)
		return
	}
	defer rows.Close()

	// Get column names from the result set.
	columns, err := rows.Columns()
	if err != nil {
		http.Error(w, "Could not get column names", http.StatusInternalServerError)
		return
	}

	// Process rows into a slice of maps for generic display.
	var results []map[string]interface{}
	for rows.Next() {
		// Create a slice of empty interfaces to hold the values for each row.
		values := make([]interface{}, len(columns))
		valuePtrs := make([]interface{}, len(columns))
		for i := range columns {
			valuePtrs[i] = &values[i]
		}

		// Scan the row into the slice of pointers.
		if err := rows.Scan(valuePtrs...); err != nil {
			http.Error(w, "Failed to scan row", http.StatusInternalServerError)
			return
		}

		// Create a map for the current row and populate it.
		rowMap := make(map[string]interface{})
		for i, colName := range columns {
			val := values[i]

			// Convert byte slices to strings for better display in HTML.
			b, ok := val.([]byte)
			if ok {
				rowMap[colName] = string(b)
			} else {
				rowMap[colName] = val
			}
		}
		results = append(results, rowMap)
	}

	// Prepare data for the template.
	data := dashboardData{
		TableName: tableName,
		Columns:   columns,
		Rows:      results,
	}

	// Execute the template.
	if err := templates.ExecuteTemplate(w, "table.html", data); err != nil {
		http.Error(w, "Could not render template", http.StatusInternalServerError)
	}
}
