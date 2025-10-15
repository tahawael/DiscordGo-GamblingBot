package main

import (
	"database/sql"
	"fmt"
	"log"
	"math"
	"math/rand"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/bwmarrin/discordgo"
)

type MinesGame struct {
	mu            sync.Mutex
	UserID        string
	Type          string
	UserName      string
	BetAmount     float64
	NumMines      int64
	Board         [4][4]bool // true = mine, false = safe
	Revealed      [4][4]bool // true = revealed, false = hidden
	SafeSpots     int
	RevealedSafe  int
	GameOver      bool
	Won           bool
	CurrentProfit float64
	StartTime     time.Time
	deferred      bool
}

// Calculate multiplier based on revealed safe spots and total mines
func calculateMultiplier(revealedSafe, totalMines int) float64 {
	totalSpots := 16.0
	mineCount := float64(totalMines)
	safeSpots := totalSpots - mineCount
	revealed := float64(revealedSafe)

	if revealed == 0 {
		return 1.0
	}

	// Calculate probability-based multiplier
	multiplier := 1.0
	for i := 0.0; i < revealed; i++ {
		remaining := totalSpots - i
		safesRemaining := safeSpots - i
		prob := safesRemaining / remaining
		multiplier /= prob
	}

	// Apply house edge (96% RTP)
	multiplier *= 1

	return math.Max(multiplier, 1.0)
}

// createMinesGame initializes a new Mines game for a user.
func createMinesGame(userID string, userName string, betAmount float64, numMines int64) *MinesGame {
	// Ensure the number of mines is valid (between 1 and 15 for a 4x4 board).
	if numMines < 1 || numMines > 15 {
		return nil
	}

	game := &MinesGame{
		UserID:        userID,
		UserName:      userName,
		Type:          "mines",
		BetAmount:     betAmount,
		NumMines:      numMines,
		SafeSpots:     16 - int(numMines),
		RevealedSafe:  0,
		GameOver:      false,
		Won:           false,
		CurrentProfit: 0.0,
		StartTime:     time.Now(),
	}

	// Initialize a 4x4 board (all spots safe and unrevealed at start). row = horizontal, col = vertical
	for row := 0; row < 4; row++ {
		for col := 0; col < 4; col++ {
			game.Board[row][col] = false    // no mine
			game.Revealed[row][col] = false // not revealed
		}
	}

	// Generate all positions
	positions := rand.Perm(16)

	for i := 0; i < int(numMines); i++ {
		row, col := positions[i]/4, positions[i]%4
		game.Board[row][col] = true
	}
	// // Debug output: show mine positions in the console.
	// for row := 0; row < 4; row++ {
	// 	for col := 0; col < 4; col++ {
	// 		if game.Board[row][col] {
	// 			fmt.Printf("💣 at (%d, %d)\n", col, row)
	// 		}
	// 	}
	// }

	return game
}

// generateMinesButtons builds the interactive Discord button grid for the Mines game
func generateMinesButtons(game *MinesGame) []discordgo.MessageComponent {
	var rows []discordgo.MessageComponent

	// Build a 4x4 grid of buttons (game board).
	for row := 0; row < 4; row++ {
		var buttons []discordgo.MessageComponent
		for col := 0; col < 4; col++ {
			// Default button state (unrevealed).
			label := "⠀"
			style := discordgo.SecondaryButton
			disabled := game.GameOver // disable everything if the game is over
			buttonID := fmt.Sprintf("mine_%s_%d_%d", game.UserID, row, col)

			// Update button appearance if the spot has been revealed.
			if game.Revealed[row][col] {
				if game.Board[row][col] { //checks if the spot is mine
					label = "💣"
					style = discordgo.DangerButton
				} else {
					label = "💎"
					style = discordgo.SuccessButton
				}
				disabled = true // disable the button after press
			}

			buttons = append(buttons, discordgo.Button{
				Label:    label,
				Style:    style,
				Disabled: disabled,
				CustomID: buttonID,
			})
		}

		rows = append(rows, discordgo.ActionsRow{
			Components: buttons,
		})
	}

	// Add extra control buttons.
	if !game.GameOver {
		rows = append(rows, discordgo.ActionsRow{
			Components: []discordgo.MessageComponent{
				discordgo.Button{
					Label:    "💰 Cash Out",
					Style:    discordgo.PrimaryButton,
					CustomID: fmt.Sprintf("cashout_%s", game.UserID),
					Disabled: game.RevealedSafe <= 0, // disabled if no safe spots revealed yet
				},
			},
		})
	} else if game.GameOver {
		// Offer a play again button once the game ends.
		rows = append(rows, discordgo.ActionsRow{
			Components: []discordgo.MessageComponent{
				discordgo.Button{
					Label:    "Play Again",
					Style:    discordgo.PrimaryButton,
					CustomID: fmt.Sprintf("playagain_%v_%v", game.BetAmount, game.NumMines),
				},
			},
		})
	}

	return rows
}

// generateGameStatus builds the status message for the Mines game.
func generateGameStatus(game *MinesGame, balance float64) string {
	var status string

	if game.GameOver {
		if game.Won {
			// Won → show "Game"
			status = fmt.Sprintf("> **%s's Won**\n", fmt.Sprintf("<@%s>", game.UserID))
		} else {
			// Lost → show "Lost"
			status = fmt.Sprintf("> **%s Lost**\n", fmt.Sprintf("<@%s>", game.UserID))
		}
	} else {
		// Still playing → show "Game"
		status = fmt.Sprintf("> **%s's Game**\n", fmt.Sprintf("<@%s>", game.UserID))
	}
	status += fmt.Sprintf("👤 Balance: %.2f\n", balance)
	status += fmt.Sprintf("💰 Bet: %.2f\n", game.BetAmount)
	status += fmt.Sprintf("💣 Mines: %d\n", game.NumMines)
	status += fmt.Sprintf("✅ Safe spots found: %d/%d\n", game.RevealedSafe, game.SafeSpots)

	if game.GameOver {
		// --- Game finished ---
		if game.Won {
			multiplier := calculateMultiplier(game.RevealedSafe, int(game.NumMines))
			profit := game.CurrentProfit
			status += fmt.Sprintf("👤 Balance: %.2f\n", balance)
			status += fmt.Sprintf("📈 Multiplier: %.2fx\n", multiplier)
			status += fmt.Sprintf("💵 Profit: +%.2f\n", profit)

			for r := 0; r < 4; r++ {
				for c := 0; c < 4; c++ {
					if game.Board[r][c] {
						game.Revealed[r][c] = true
					}
				}
			}

		} else {
			// Player lost → show bet loss and update DB.
			status += fmt.Sprintf("💸 Loss: %.2f\n", game.BetAmount)
			logGame(game.UserID, "mines", game.BetAmount, -game.BetAmount)

		}

	} else {
		// --- Game still in progress ---
		multiplier := calculateMultiplier(game.RevealedSafe, int(game.NumMines))
		// profit := float64(game.BetAmount) * (multiplier - 1)
		profit := game.CurrentProfit

		status += fmt.Sprintf("📈 Multiplier: %.2fx\n", multiplier)
		status += fmt.Sprintf("💵 Potential profit: +%.2f\n", profit)
	}
	return status
}

// messageComponentHandlers handles button and select menu interactions for the Mines game.
func handleBtns(s *discordgo.Session, i *discordgo.InteractionCreate) {

	// startTime := time.Now() // Start timing

	if i.Type != discordgo.InteractionMessageComponent {
		return
	}
	customID := i.MessageComponentData().CustomID

	// Only enforce "active game required" for actions that actually need it.
	// Safely get user ID for ban check
	var userID string
	if i.Member != nil && i.Member.User != nil {
		userID = i.Member.User.ID
	} else if i.User != nil {
		userID = i.User.ID
	} else {
		// Log this case and return - shouldn't happen normally
		log.Printf("Warning: Could not determine user ID for interaction in guild %s", i.GuildID)
		return
	}
	if banChk(s, i, userID) {
		return // stop here if banned
	}
	var userBalance float64
	err := db.QueryRow("SELECT balance FROM users WHERE userid = ?", userID).Scan(&userBalance)
	if err == sql.ErrNoRows {
		if err := respondEphemeral(s, i, "❌ You're not registered! registring user...", nil); err != nil {
			log.Println("respondUpdate error (not registered):", err)
		}
		addUser(s, i)
		return
	} else if err != nil {
		log.Println("DB error checking balance:", err)
		if err := respondEphemeral(s, i, "❌ Database error!", nil); err != nil {
			log.Println("respondUpdate error (DB error):", err)
		}
		return
	}
	if strings.HasPrefix(customID, "playagainSlot_") {
		parts := strings.Split(customID, "_")
		if len(parts) >= 2 {
			amountStr := parts[1]
			betAmount, err := strconv.ParseFloat(amountStr, 64)
			if err == nil {
				// Restart the slot game with same bet amount
				slot(s, i, betAmount)
			}
		}
		return
	}
	if strings.HasPrefix(customID, "daily_claim") {
		HandleDailyClaimButton(s, i, db, userID)
		return
	}

	mineGame, err := getActiveGameFromDB(userID)
	if err != nil {
		// No active game found in DB
		if !strings.HasPrefix(customID, "playagain_") {
			respondEphemeral(s, i, "❌ You don't have an active mines game!", nil)
			return
		}
		// if "playagain_*", just continue with mineGame == nil
	}

	// If there is a game and it is over, block further interactions except "Play Again".
	if mineGame != nil && mineGame.GameOver && !strings.HasPrefix(customID, "playagain_") {
		respondEphemeral(s, i, "❌ This game has already ended!", nil)
		return
	}

	done := make(chan struct{})
	if mineGame != nil {
		game := mineGame // capture
		go func() {
			timer := time.NewTimer(2500 * time.Millisecond)
			defer timer.Stop()

			select {
			case <-timer.C:
				game.mu.Lock()
				game.deferred = true
				game.mu.Unlock()
			case <-done:
				// Timer cancelled, no need to set deferred
			}
		}()
	}

	// ticker := time.NewTicker(2500 * time.Millisecond)
	// done := make(chan bool)  // This should be closed when the interaction is done (e.g., game over)
	// startTimeD := time.Now() // Start timing

	// go func() {
	// 	for {
	// 		select {
	// 		case <-ticker.C:
	// 			// fmt.Println("Tick at second:", t.Second())
	// 			log.Printf("Dfferdede %v\n", time.Since(startTimeD))

	// 			// Send deferred update
	// 			err := s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
	// 				Type: discordgo.InteractionResponseDeferredMessageUpdate,
	// 			})
	// 			if err != nil {
	// 				log.Println("Deferred update failed:", err)
	// 				continue // Or break, depending on your error handling policy
	// 			}

	// 			// Lock to safely update game.deferred
	// 			mineGame.mu.Lock()
	// 			mineGame.deferred = true
	// 			mineGame.mu.Unlock()

	// 		case <-done:
	// 			ticker.Stop()
	// 			fmt.Println("Ticker stopped")
	// 			return
	// 		}
	// 	}
	// }()

	// When you want to stop the ticker later
	switch {
	case strings.HasPrefix(customID, "cashout_"):
		handleCashout(s, i, mineGame, userID, customID, userBalance)
	case strings.HasPrefix(customID, "playagain_"):
		handlePlayAgain(s, i, userID, userBalance)
	case strings.HasPrefix(customID, "mine_"):
		handleMineClick(s, i, mineGame, customID, userBalance)
	}
	// done <- true
	close(done) // cancels if within 2.5s
	// log.Printf("generateGameStatus hit mine %v\n", time.Since(startTime))

}

func startMinesGame(s *discordgo.Session, i *discordgo.InteractionCreate, userID string, betAmount float64, numMines int64, balance float64) {
	if banChk(s, i, userID) {
		return // stop here if banned
	}
	// Check user balance

	// startTime := time.Now() // Start timing
	// If user already has an active game, show it instead of starting a new one
	if game, err := getActiveGameFromDB(userID); err == nil && game.Type == "mines" {
		status := generateGameStatus(game, balance)
		buttons := generateMinesButtons(game)
		if err := respondEphemeral(s, i, "🟢 You already have an active game! Continue playing:\n\n"+status, buttons); err != nil {
			log.Println("respondUpdate error (active game):", err)
		}
		return
	}

	// Validate bet amount
	if betAmount <= 0 {
		if err := respondEphemeral(s, i, "❌ Bet amount must be greater than 0!", nil); err != nil {
			log.Println("respondUpdate error (invalid bet):", err)
		}
		return
	}

	// Validate mines count
	if numMines < 1 || numMines > 15 {
		if err := respondEphemeral(s, i, "❌ Number of mines must be between 1 and 15!", nil); err != nil {
			log.Println("respondUpdate error (invalid mines):", err)
		}
		return
	}

	// Balance check
	if balance < betAmount {
		if err := respondEphemeral(s, i, "❌ Insufficient balance!", nil); err != nil {
			log.Println("respondUpdate error (insufficient balance):", err)
		}
		return
	}
	var username string
	if i.Member != nil && i.Member.User != nil {
		username = i.Member.User.Username
	} else if i.User != nil {
		username = i.User.Username
	} else {
		// Log this case and return - shouldn't happen normally
		log.Printf("Warning: Could not determine username for interaction in guild %s", i.GuildID)
		return
	}
	// Create new game
	game := createMinesGame(userID, username, betAmount, numMines)
	if game == nil {
		if err := respondEphemeral(s, i, "❌ Failed to create game!", nil); err != nil {
			log.Println("respondUpdate error (create game):", err)
		}
		return
	}

	// Save DB in background (non-blocking)
	if err := saveActiveGameToDB(game); err != nil {
		log.Println("Error saving game:", err)
	}

	// Respond with new game state
	if err := sendNewMessage(s, i, generateGameStatus(game, balance), generateMinesButtons(game)); err != nil {
		log.Println("respondUpdate error (new game):", err)
	}
	// log.Printf("startMinesGame %v\n", time.Since(startTime))

}

func handleCashout(s *discordgo.Session, i *discordgo.InteractionCreate, game *MinesGame, userID string, customID string, balance float64) {
	parts := strings.Split(customID, "_")
	if len(parts) < 2 {
		return
	}

	ownerID := parts[1]

	// Block if not the game owner
	if userID != ownerID {
		respondEphemeral(s, i, "❌ This isn't your game!", nil)
		return
	}

	multiplier := calculateMultiplier(game.RevealedSafe, int(game.NumMines))
	winAmount := float64(game.BetAmount) * (multiplier - 1)

	// // Update balance
	// if _, err := db.Exec("UPDATE users SET balance = ROUND(balance + ?, 2) WHERE userid = ?", winAmount, userID); err != nil {
	// 	log.Println("DB error on cashout:", err)
	// 	respondEphemeral(s, i, "❌ Error processing cashout!", nil)
	// 	return
	// }
	// logGame(game.UserID, "mines", game.BetAmount, winAmount)
	// // Reveal all mines
	// for r := 0; r < 4; r++ {
	// 	for c := 0; c < 4; c++ {
	// 		if game.Board[r][c] {
	// 			game.Revealed[r][c] = true
	// 		}
	// 	}
	// }
	// // Update stats
	// if _, err := db.Exec("UPDATE users SET wins = ROUND(wins + ?, 2) WHERE userid = ?", winAmount, userID); err != nil {
	// 	log.Println("DB error updating wins:", err)
	// }

	// Start a transaction
	tx, err := db.Begin()

	if err != nil {
		log.Println("DB error starting transaction:", err)
		respondEphemeral(s, i, "❌ Error processing cashout!", nil)
		return
	}
	defer tx.Rollback() // rollback on failure

	// Update balance
	if _, err := tx.Exec(
		"UPDATE users SET balance = ROUND(balance + ?, 2) WHERE userid = ?",
		winAmount, userID,
	); err != nil {
		log.Println("DB error on cashout:", err)
		respondEphemeral(s, i, "❌ Error processing cashout!", nil)
		return
	}

	// Update stats (wins)
	if _, err := tx.Exec(
		"UPDATE users SET wins = ROUND(wins + ?, 2) WHERE userid = ?",
		winAmount, userID,
	); err != nil {
		log.Println("DB error updating wins:", err)
		return
	}

	// Commit transaction (only applies if both queries succeed)
	if err := tx.Commit(); err != nil {
		log.Println("DB error committing transaction:", err)
		respondEphemeral(s, i, "❌ Error processing cashout!", nil)
		return
	}

	// Logging + game state updates happen after successful commit
	logGame(game.UserID, "mines", game.BetAmount, winAmount)

	// Reveal all mines
	for r := 0; r < 4; r++ {
		for c := 0; c < 4; c++ {
			if game.Board[r][c] {
				game.Revealed[r][c] = true
			}
		}
	}

	// Mark game over
	game.GameOver = true
	game.Won = true

	deleteActiveGameFromDB(userID, game.Type)
	status := fmt.Sprintf(
		"> **%s CASHED OUT!**\n"+
			"👤 Balance: %.2f\n"+
			"💰 Bet: %.2f\n"+
			"💣 Mines: %d\n"+
			"✅ Safe spots found: %d/%d\n"+
			"📈 Multiplier: %.2fx\n"+
			"💵 Profit: +%.2f\n",
		fmt.Sprintf("<@%s>", userID), balance+winAmount, game.BetAmount, game.NumMines, game.RevealedSafe, game.SafeSpots, multiplier, winAmount,
	)

	respondUpdate(s, i, status, generateMinesButtons(game), game)
}

// handlePlayAgain handles the "Play Again" button click.
// The button's CustomID must be in the format: playagain_<betAmount>_<numMines>
func handlePlayAgain(s *discordgo.Session, i *discordgo.InteractionCreate, userID string, balance float64) {
	parts := strings.Split(i.MessageComponentData().CustomID, "_")
	if len(parts) != 3 {
		respondEphemeral(s, i, "❌ Invalid play again button data!", nil)
		return
	}

	var betAmount float64
	var numMines int64
	if _, err := fmt.Sscanf(parts[1]+" "+parts[2], "%f %d", &betAmount, &numMines); err != nil {
		log.Println("Error parsing playagain CustomID:", err)
		respondEphemeral(s, i, "❌ Invalid play again data format!", nil)
		return
	}

	startMinesGame(s, i, userID, betAmount, numMines, balance)
}

func handleMineClick(s *discordgo.Session, i *discordgo.InteractionCreate, game *MinesGame, customID string, balance float64) {
	// Parse button ID: "mine_<row>_<col>"

	parts := strings.Split(customID, "_")
	if len(parts) < 4 {
		return
	}

	ownerID := parts[1] // the player who owns this game
	row, _ := strconv.Atoi(parts[2])
	col, _ := strconv.Atoi(parts[3])
	var userID string
	if i.Member != nil && i.Member.User != nil {
		userID = i.Member.User.ID
	} else if i.User != nil {
		userID = i.User.ID
	} else {
		// Log this case and return - shouldn't happen normally
		log.Printf("Warning: Could not determine user ID for interaction in guild %s", i.GuildID)
		return
	}
	// Block clicks from non-owners
	if userID != ownerID {
		respondEphemeral(s, i, "❌ This isn't your game!", nil)
		return
	}

	// Validate coordinates
	if row < 0 || row >= 4 || col < 0 || col >= 4 {
		log.Printf("Out-of-range mine coordinates: row=%d col=%d", row, col)
		respondEphemeral(s, i, "❌ Invalid tile!", nil)
		return
	}

	// Already revealed
	if game.Revealed[row][col] {
		respondEphemeral(s, i, "❌ This tile is already revealed!", nil)
		return
	}
	// s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
	// 	Type: discordgo.InteractionResponseDeferredMessageUpdate,
	// })
	game.Revealed[row][col] = true

	// // --- Case 1: Hit a mine ---
	// if game.Board[row][col] {
	// 	game.GameOver, game.Won = true, false

	// 	for r := 0; r < 4; r++ {
	// 		for c := 0; c < 4; c++ {
	// 			if game.Board[r][c] {
	// 				game.Revealed[r][c] = true
	// 			}
	// 		}
	// 	}
	// 	if err := respondUpdate(s, i, generateGameStatus(game), generateMinesButtons(game), game); err != nil {
	// 		log.Println("respondUpdate error (hit mine):", err)
	// 	}
	// 	// Handle DB
	// 	if _, err := db.Exec(
	// 		`UPDATE users
	// 			SET balance = ROUND(balance - ?, 2),
	// 			 losses  = ROUND(losses + ?, 2)
	// 		 WHERE userid = ?`,
	// 		game.BetAmount, game.BetAmount, game.UserID,
	// 	); err != nil {
	// 		log.Println("DB error updating balance and losses:", err)
	// 	}

	// 	if err := deleteActiveGameFromDB(game.UserID); err != nil {
	// 		log.Printf("DB error deleting game for user %s: %v", game.UserID, err)
	// 	}

	// 	return
	// }
	// --- Case 1: Hit a mine ---
	if game.Board[row][col] {
		game.GameOver, game.Won = true, false

		// --- DB handling with transaction ---
		tx, err := db.Begin()
		if err != nil {
			log.Println("DB error starting transaction:", err)
			return
		}
		defer tx.Rollback()

		// Update balance and losses
		if _, err := tx.Exec(
			`UPDATE users
			SET balance = ROUND(balance - ?, 2),
			    losses  = ROUND(losses + ?, 2)
		 WHERE userid = ?`,
			game.BetAmount, game.BetAmount, game.UserID,
		); err != nil {
			log.Println("DB error updating balance and losses:", err)
			return
		}

		// Delete active game
		if err := deleteActiveGameFromDBTx(tx, game.UserID, "mines"); err != nil {
			log.Printf("DB error deleting game for user %s: %v", game.UserID, err)
			return
		}

		// Commit if both succeed
		if err := tx.Commit(); err != nil {
			log.Println("DB error committing transaction:", err)
			return
		}
		// Reveal all mines
		for r := 0; r < 4; r++ {
			for c := 0; c < 4; c++ {
				if game.Board[r][c] {
					game.Revealed[r][c] = true
				}
			}
		}

		// First try to update UI
		if err := respondUpdate(s, i, generateGameStatus(game, balance-game.BetAmount), generateMinesButtons(game), game); err != nil {
			log.Println("respondUpdate error (hit mine):", err)
		}

		return
	}

	// time.Sleep(3 * time.Second)
	// --- Case 2: Safe tile ---
	game.RevealedSafe++
	multiplier := calculateMultiplier(game.RevealedSafe, int(game.NumMines))
	game.CurrentProfit = float64(game.BetAmount) * (multiplier - 1)

	if game.RevealedSafe >= game.SafeSpots {
		// Auto-win
		game.GameOver, game.Won = true, true
		winAmount := game.CurrentProfit

		// Handle DB first (synchronously for security)
		logGame(game.UserID, "mines", game.BetAmount, winAmount)

		// if _, err := db.Exec(
		// 	"UPDATE users SET wins = ROUND(wins + ?, 2) WHERE userid = ?",
		// 	winAmount, game.UserID,
		// ); err != nil {
		// 	log.Println("DB error adding wins:", err)
		// }

		// if _, err := db.Exec(
		// 	"UPDATE users SET balance = ROUND(balance + ?, 2) WHERE userid = ?",
		// 	winAmount, game.UserID,
		// ); err != nil {
		// 	log.Println("DB error updating balance:", err)
		// }

		// if err := deleteActiveGameFromDB(game.UserID); err != nil {
		// 	log.Println("DB error deleting game:", err)
		// }
		// // Respond only after DB is done
		// if err := respondUpdate(s, i, generateGameStatus(game), generateMinesButtons(game), game); err != nil {
		// 	log.Println("respondUpdate error (auto-win):", err)
		// }
		tx, err := db.Begin()
		if err != nil {
			log.Println("DB error starting transaction:", err)
			return
		}
		defer tx.Rollback()

		// Update wins
		if _, err := tx.Exec(
			"UPDATE users SET wins = ROUND(wins + ?, 2) WHERE userid = ?",
			winAmount, game.UserID,
		); err != nil {
			log.Println("DB error adding wins:", err)
			return
		}

		// Update balance
		if _, err := tx.Exec(
			"UPDATE users SET balance = ROUND(balance + ?, 2) WHERE userid = ?",
			winAmount, game.UserID,
		); err != nil {
			log.Println("DB error updating balance:", err)
			return
		}

		// Delete active game
		if err := deleteActiveGameFromDBTx(tx, game.UserID, "mines"); err != nil {
			log.Println("DB error deleting game:", err)
			return
		}

		// Commit if all succeed
		if err := tx.Commit(); err != nil {
			log.Println("DB error committing transaction:", err)
			return
		}

		// Respond only after DB is fully committed
		if err := respondUpdate(s, i, generateGameStatus(game, balance+winAmount), generateMinesButtons(game), game); err != nil {
			log.Println("respondUpdate error (auto-win):", err)
		}

		return
	}

	// Save DB in background
	if err := saveActiveGameToDB(game); err != nil {
		log.Println("Error saving game:", err)
	}
	// time.Sleep(5 * time.Second)
	// startTime := time.Now() // Start timing

	// --- Case 3: Normal safe tile (game continues) ---
	if err := respondUpdate(s, i, generateGameStatus(game, balance), generateMinesButtons(game), game); err != nil {
		log.Println("respondUpdate error (safe tile):", err)
	}
	// log.Printf("generateGameStatushit safe %v\n", time.Since(startTime))

}
