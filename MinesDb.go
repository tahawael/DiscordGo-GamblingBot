package main

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"time"
)

func saveActiveGameToDB(game *MinesGame) error {
	// startTime := time.Now() // Start timing
	boardJSON, _ := json.Marshal(game.Board)
	revealedJSON, _ := json.Marshal(game.Revealed)

	_, err := db.Exec(`
        INSERT INTO active_games (userid,type,username, bet_amount, num_mines, board, revealed, 
                                safe_spots, revealed_safe, game_over, won, current_profit)
        VALUES (?, ?,?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
        ON DUPLICATE KEY UPDATE
			bet_amount     = IF(type = VALUES(type), VALUES(bet_amount), bet_amount),
			board          = IF(type = VALUES(type), VALUES(board), board),
			revealed       = IF(type = VALUES(type), VALUES(revealed), revealed),
			revealed_safe  = IF(type = VALUES(type), VALUES(revealed_safe), revealed_safe),
			game_over      = IF(type = VALUES(type), VALUES(game_over), game_over),
			won            = IF(type = VALUES(type), VALUES(won), won),
			current_profit = IF(type = VALUES(type), VALUES(current_profit), current_profit)`,

		game.UserID, game.Type, game.UserName, game.BetAmount, game.NumMines, boardJSON, revealedJSON,
		game.SafeSpots, game.RevealedSafe, game.GameOver, game.Won, game.CurrentProfit)
	return err
}

func saveActiveslotToDB(UserId string) error {

	_, err := db.Exec(`
        INSERT INTO active_games (userid,type,username, bet_amount, num_mines, board, revealed, 
    safe_spots, revealed_safe, game_over, won, current_profit)
        VALUES (?, ?, ?,0.00, 0, '[]', '[]', 0, 0, FALSE, FALSE, 0.00)`,
		UserId, "slot", "")
	return err

}

func getActiveGameFromDB(userID string) (*MinesGame, error) {
	// startTime := time.Now() // Start timing

	var game MinesGame
	var boardJSON, revealedJSON string

	// log.Printf("DEBUG: Querying database for user %s", userID)
	// Remove start_time from the SELECT query
	err := db.QueryRow(`
        SELECT userid, type, username, bet_amount, num_mines, board, revealed, safe_spots, 
               revealed_safe, game_over, won, current_profit
        FROM active_games WHERE userid = ?`, userID).Scan(
		&game.UserID, &game.Type, &game.UserName, &game.BetAmount, &game.NumMines, &boardJSON, &revealedJSON,
		&game.SafeSpots, &game.RevealedSafe, &game.GameOver, &game.Won,
		&game.CurrentProfit) // Remove &game.StartTime

	if err != nil {
		// log.Printf("DEBUG: Database query error: %v", err)
		return nil, err
	}

	// Set StartTime to current time since we're not storing it
	// game.StartTime = time.Now()

	// log.Printf("DEBUG: Raw data from DB - boardJSON: %s, revealedJSON: %s", boardJSON, revealedJSON)

	if err := json.Unmarshal([]byte(boardJSON), &game.Board); err != nil {
		// log.Printf("DEBUG: Board JSON unmarshal error: %v", err)
		return nil, err
	}

	if err := json.Unmarshal([]byte(revealedJSON), &game.Revealed); err != nil {
		// log.Printf("DEBUG: Revealed JSON unmarshal error: %v", err)
		return nil, err
	}

	// log.Printf("DEBUG: Successfully loaded game for user %s", userID)
	// log.Printf("getGameFromDB %v\n", time.Since(startTime))
	return &game, nil
}

func deleteActiveGameFromDB(userID string, Type string) error {
	for i := 0; i < 3; i++ {
		_, err := db.Exec("DELETE FROM active_games WHERE userid = ? AND type = ?", userID, Type)
		if err == nil {
			return nil
		}
		log.Printf("retry delete attempt %d failed: %v", i+1, err)
		time.Sleep(50 * time.Millisecond)
	}
	return fmt.Errorf("failed to delete active game for %s after retries", userID)
}
func deleteActiveGameFromDBTx(tx *sql.Tx, userID string, Type string) error {
	for i := 0; i < 3; i++ {
		_, err := db.Exec("DELETE FROM active_games WHERE userid = ?  AND type = ?", userID, Type)
		if err == nil {
			return nil
		}
		log.Printf("retry delete attempt %d failed: %v", i+1, err)
		time.Sleep(50 * time.Millisecond)
	}
	return fmt.Errorf("failed to delete active game for %s after retries", userID)
}

func logGame(userID string, game_type string, amount float64, outcome float64) {
	// log.Println("starting logGame")
	_, err := db.Exec(
		"INSERT INTO games (userid, game_type, amount, outcome) VALUES (?, ?, ROUND(?, 2), ROUND(?, 2))",
		userID, game_type, amount, outcome,
	)
	if err != nil {
		log.Println("Error logging game:", err)
	}
	// log.Println("ending logGame")
}
func logTrans(sender int, sendername string, receiver int, receivername string, amount float64, status string) {
	// log.Println("starting logGame")
	_, err := db.Exec(
		"INSERT INTO transactions (sender, sendername, receiver, receivername, amount, status) VALUES (?, ?, ?, ?, ROUND(?, 2), ?)",
		sender, sendername, receiver, receivername, amount, status,
	)
	if err != nil {
		log.Println("Error logging transaction:", err)
	}
	// log.Println("ending logGame")
}
