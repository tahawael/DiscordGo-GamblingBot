package main

import (
	"database/sql"
	"fmt"
	"log"
	"math/rand"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/bwmarrin/discordgo"
	_ "github.com/go-sql-driver/mysql"
)

var db *sql.DB

// Setup all tables with proper schema
func setupTables(db *sql.DB) error {
	// Define all table schemas
	tables := map[string]string{
		"users": `CREATE TABLE IF NOT EXISTS users (
			userid BIGINT UNSIGNED PRIMARY KEY,
			username VARCHAR(32) NOT NULL,
			balance DECIMAL(10,2) NOT NULL DEFAULT 0.00,
			wins DECIMAL(10,2) NOT NULL DEFAULT 0.00,
			losses DECIMAL(10,2) NOT NULL DEFAULT 0.00,
			admin TINYINT NOT NULL DEFAULT 0,
			banned TINYINT NOT NULL DEFAULT 0
		)`,

		"active_games": `CREATE TABLE IF NOT EXISTS active_games (
			id INT AUTO_INCREMENT PRIMARY KEY,
			userid BIGINT UNSIGNED ,
			type VARCHAR(32) NOT NULL ,
			username VARCHAR(32) NOT NULL,
			bet_amount DECIMAL(10,2) NOT NULL,
			num_mines INT NOT NULL,
			board JSON NOT NULL,
			revealed JSON NOT NULL,
			safe_spots INT NOT NULL,
			revealed_safe INT NOT NULL,
			game_over BOOLEAN NOT NULL DEFAULT FALSE,
			won BOOLEAN NOT NULL DEFAULT FALSE,
			current_profit DECIMAL(10,2) NOT NULL DEFAULT 0.00,
			start_time TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
			last_updated TIMESTAMP DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
			UNIQUE KEY (userid, type),
			FOREIGN KEY (userid) REFERENCES users(userid) ON DELETE CASCADE,
		)`,

		"games": `CREATE TABLE IF NOT EXISTS games (
			id INT AUTO_INCREMENT PRIMARY KEY,
			userid BIGINT UNSIGNED NOT NULL,
			game_type VARCHAR(32) NOT NULL,
			amount DECIMAL(10,2) NOT NULL,
			outcome DECIMAL(10,2) NOT NULL,
			played_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
			FOREIGN KEY (userid) REFERENCES users(userid) ON DELETE CASCADE
		)`,

		"transactions": `CREATE TABLE IF NOT EXISTS transactions (
			id INT AUTO_INCREMENT PRIMARY KEY,
			sender BIGINT UNSIGNED NOT NULL,
			sendername VARCHAR(32) NOT NULL,
			receiver BIGINT UNSIGNED NOT NULL,
			receivername VARCHAR(32) NOT NULL,
			amount DECIMAL(12,2) NOT NULL,
			status VARCHAR(32) NOT NULL,
			played_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
			FOREIGN KEY (sender) REFERENCES users(userid) ON DELETE CASCADE,
			FOREIGN KEY (receiver) REFERENCES users(userid) ON DELETE CASCADE
		)`,
		"daily_rewards": `CREATE TABLE IF NOT EXISTS daily_rewards (
			id INT AUTO_INCREMENT PRIMARY KEY,
			userid BIGINT UNSIGNED NOT NULL,
			claim_date DATE NOT NULL,
			streak INT NOT NULL,
			reward_amount DECIMAL(10,2) NOT NULL,
			claimed_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
			FOREIGN KEY (userid) REFERENCES users(userid) ON DELETE CASCADE,
			UNIQUE(userid, claim_date) -- ensures 1 claim per day
		)`,
	}

	// Create tables in order (users first, then dependent tables)
	order := []string{"users", "active_games", "games", "transactions", "daily_rewards"}

	for _, tableName := range order {
		if _, err := db.Exec(tables[tableName]); err != nil {
			log.Printf("Error creating table %s: %v", tableName, err)
			return err
		}
		log.Printf("Table %s created/verified successfully", tableName)
	}

	return nil
}

// ALTER TABLE users ADD COLUMN banned TINYINT NOT NULL DEFAULT 0;
func main() {
	var err error
	rand.New(rand.NewSource(time.Now().UnixNano()))

	dbPath := os.Getenv("dbPath")
	gamblingBotToken := os.Getenv("gamblingBotToken")

	fmt.Println("dbPath from env:", dbPath)
	fmt.Println("gamblingBotToken from env:", gamblingBotToken)

	db, err = sql.Open("mysql", dbPath)
	if err != nil {
		log.Fatal("Failed to open database:", err)
	}

	// verify connection
	if err = db.Ping(); err != nil {
		log.Fatal("Database not reachable:", err)
	}

	// Add these for better connection management
	db.SetMaxOpenConns(500) // max connections MySQL can handle
	db.SetMaxIdleConns(50)  // keep some idle connections ready
	db.SetConnMaxLifetime(5 * time.Minute)
	_, err = db.Exec(`SET GLOBAL event_scheduler = ON;`)
	if err != nil {
		log.Printf("Failed to enable event scheduler: %v", err)
	}

	// Create event to auto-delete inactive games
	_, err = db.Exec(`
    CREATE EVENT IF NOT EXISTS delete_inactive_games
    ON SCHEDULE EVERY 1 MINUTE
    DO
      DELETE FROM active_games
      WHERE last_updated < NOW() - INTERVAL 5 MINUTE;
`)
	if err != nil {
		log.Printf("Failed to create event: %v", err)
	}

	log.Println("Database connected successfully")

	// only run when setting up the db
	// if err := setupTables(db); err != nil {
	// 	log.Fatal("Failed to setup tables:", err)
	// }

	defer db.Close()
	// Create a new Discord session using the provided bot token.
	dg, err := discordgo.New("Bot " + gamblingBotToken)
	if err != nil {
		fmt.Println("error creating Discord session,", err)
		return
	}

	// Add handlers
	dg.AddHandler(interactionCreate)
	dg.AddHandler(handleBtns)

	// Set intents
	dg.Identify.Intents = discordgo.IntentsGuilds

	// Open connection
	err = dg.Open()
	if err != nil {
		fmt.Println("error opening connection,", err)
		return
	}
	addCommands(dg)
	StartDashboard(db, "8080")

	log.Printf("Bot running. Press CTRL-C to exit.")

	stop := make(chan os.Signal, 1)
	signal.Notify(stop, syscall.SIGINT, syscall.SIGTERM, os.Interrupt)
	<-stop
	defer db.Close()
	defer dg.Close()
}

