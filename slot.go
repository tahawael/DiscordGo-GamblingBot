package main

import (
	"context"
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

// Global emoji cache for multiple guilds
var guildEmojiCache = make(map[string]map[string]string) // guildID -> emoji_name -> formatted_emoji
var cacheInitialized = make(map[string]bool)             // guildID -> initialized
var cacheMutex sync.Mutex

// servers that have the gifs uploaded to them that the bot must join (you can use one if you have enough slots)
var guildIDs = []string{"", ""}

// Initialize emoji cache for a specific guild
func initEmojiCache(s *discordgo.Session, guildID string) error {
	guild, err := s.Guild(guildID)
	if err != nil {
		return fmt.Errorf("failed to get guild %s: %w", guildID, err)
	}

	// Initialize guild cache if it doesn't exist
	if guildEmojiCache[guildID] == nil {
		guildEmojiCache[guildID] = make(map[string]string)
	}

	// Clear existing cache for this guild
	guildEmojiCache[guildID] = make(map[string]string)

	// Populate cache with all guild emojis
	for _, emoji := range guild.Emojis {
		if emoji.Animated {
			guildEmojiCache[guildID][emoji.Name] = fmt.Sprintf("<a:%s:%s>", emoji.Name, emoji.ID)
		} else {
			guildEmojiCache[guildID][emoji.Name] = fmt.Sprintf("<:%s:%s>", emoji.Name, emoji.ID)
		}
	}

	fmt.Printf("Loaded %d emojis into cache for guild %s\n", len(guildEmojiCache[guildID]), guildID)
	cacheInitialized[guildID] = true
	return nil
}

// Helper function to get emoji from cache with fallback and auto-initialization
// Tries multiple guilds in order until emoji is found
func getEmoji(name string, fallback string, s *discordgo.Session, guildIDs []string) string {
	cacheMutex.Lock()
	defer cacheMutex.Unlock()

	for _, guildID := range guildIDs {
		// Initialize cache if not already done for this guild
		if !cacheInitialized[guildID] {
			err := initEmojiCache(s, guildID)
			if err != nil {
				fmt.Printf("Failed to initialize emoji cache for guild %s: %v\n", guildID, err)
				continue // Try next guild
			}
		}

		// Check if emoji exists in this guild
		if guildCache, exists := guildEmojiCache[guildID]; exists {
			if emoji, emojiExists := guildCache[name]; emojiExists {
				return emoji
			}
		}
	}

	return fallback // Use fallback if emoji not found in any guild
}

// Build emoji grid with 3 different tile numbers
func buildEmojiGrid(input1, input2, input3 string, s *discordgo.Session) string {
	rows := []string{}
	for row := 0; row < 3; row++ {
		rowEmojis := ""
		// First pair (input1)
		rowEmojis += getEmoji(fmt.Sprintf("%s_rounded_tile_%d_0", input1, row), ":question:", s, guildIDs)
		rowEmojis += getEmoji(fmt.Sprintf("%s_rounded_tile_%d_1", input1, row), ":question:", s, guildIDs)
		rowEmojis += " " // Space between pairs
		// Second pair (input2)
		rowEmojis += getEmoji(fmt.Sprintf("%s_rounded_tile_%d_0", input2, row), ":question:", s, guildIDs)
		rowEmojis += getEmoji(fmt.Sprintf("%s_rounded_tile_%d_1", input2, row), ":question:", s, guildIDs)
		rowEmojis += " " // Space between pairs
		// Third pair (input3)
		rowEmojis += getEmoji(fmt.Sprintf("%s_rounded_tile_%d_0", input3, row), ":question:", s, guildIDs)
		rowEmojis += getEmoji(fmt.Sprintf("%s_rounded_tile_%d_1", input3, row), ":question:", s, guildIDs)
		rows = append(rows, rowEmojis)
	}
	return strings.Join(rows, "\n")
}

func buildEmbed(description string, color int, betAmount float64, payout *float64, balance float64) *discordgo.MessageEmbed {
	fields := []*discordgo.MessageEmbedField{
		{
			Name:   "💰 Bet Amount",
			Value:  fmt.Sprintf("`$%.2f`", betAmount),
			Inline: true,
		},
	}

	// Add Status field only on final reveal
	var status string
	status = "`$0.00`"
	if payout != nil && *payout > 0 {
		winAmount := *payout * betAmount
		status = fmt.Sprintf("`$%.2f`\n", winAmount)
	}

	fields = append(fields, &discordgo.MessageEmbedField{
		Name:   "📊 Win",
		Value:  status,
		Inline: true,
	})
	fields = append(fields, &discordgo.MessageEmbedField{
		Value:  fmt.Sprintf("👤 Balance `$%.2f`", balance),
		Inline: false,
	})

	return &discordgo.MessageEmbed{
		Title:       "🎰 Slot Machine 🎰",
		Description: fmt.Sprintf("**%s**", description),
		Color:       color,
		Fields:      fields,
		// Footer: &discordgo.MessageEmbedFooter{
		// 	Text: fmt.Sprintf("👤 Played by %s", username),
		// },
	}
}

func sendAnimatedEmojiGridBatched(
	s *discordgo.Session,
	i *discordgo.InteractionCreate, // use interaction instead of channelID
	maxSymbol int,
	embedTitle string,
	embedColor int) {
	cacheMutex.Lock()
	defer cacheMutex.Unlock()

	var allGrids []string

	// Collect all grids from all guilds first
	for _, guildID := range guildIDs {
		if !cacheInitialized[guildID] {
			err := initEmojiCache(s, guildID)
			if err != nil {
				fmt.Printf("Failed to initialize emoji cache for guild %s: %v\n", guildID, err)
				continue
			}
		}

		if guildCache, exists := guildEmojiCache[guildID]; exists {
			for symbol := 0; symbol < maxSymbol; symbol++ {
				rows := []string{}
				for row := 0; row < 3; row++ {
					rowEmojis := ""
					for col := 0; col < 2; col++ {
						emojiName := fmt.Sprintf("%d_rounded_tile_%d_%d", symbol, row, col)
						if emoji, ok := guildCache[emojiName]; ok {
							if strings.HasPrefix(emoji, "<a:") {
								rowEmojis += emoji
							}
						}
					}
					if rowEmojis != "" {
						rows = append(rows, rowEmojis)
					}
				}
				if len(rows) > 0 {
					grid := strings.Join(rows, "\n")
					allGrids = append(allGrids, grid)
				}
			}
		}
	}

	// Send in exactly 2 embeds with 6 grids each
	gridsPerEmbed := 6
	var embeds []*discordgo.MessageEmbed

	for embedIndex := 0; embedIndex < 2; embedIndex++ {
		startIdx := embedIndex * gridsPerEmbed
		endIdx := startIdx + gridsPerEmbed
		if startIdx >= len(allGrids) {
			break
		}
		if endIdx > len(allGrids) {
			endIdx = len(allGrids)
		}

		var messageRows []string
		maxRows := 0
		batchGrids := allGrids[startIdx:endIdx]

		for _, grid := range batchGrids {
			gridRows := strings.Split(grid, "\n")
			if len(gridRows) > maxRows {
				maxRows = len(gridRows)
			}
		}

		for rowIdx := 0; rowIdx < maxRows; rowIdx++ {
			var rowParts []string
			for _, grid := range batchGrids {
				gridRows := strings.Split(grid, "\n")
				if rowIdx < len(gridRows) {
					rowParts = append(rowParts, gridRows[rowIdx])
				} else {
					rowParts = append(rowParts, "")
				}
			}
			messageRows = append(messageRows, strings.Join(rowParts, " "))
		}

		gridContent := strings.Join(messageRows, "\n")
		if gridContent != "" {
			embed := &discordgo.MessageEmbed{
				Title:       fmt.Sprintf("%s - Page %d", embedTitle, embedIndex+1),
				Description: gridContent,
				Color:       embedColor,
				Footer: &discordgo.MessageEmbedFooter{
					Text: fmt.Sprintf("Loading Grids %d-%d", startIdx+1, endIdx),
				},
			}
			embeds = append(embeds, embed)
		}
	}

	// Respond ephemerally
	err := s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{
			Flags:  discordgo.MessageFlagsEphemeral,
			Embeds: embeds,
		},
	})

	if err != nil {
		fmt.Printf("Failed to send ephemeral emoji grids: %v\n", err)
	}
}

func slot(s *discordgo.Session, i *discordgo.InteractionCreate, betAmount float64) {
	// Send a deferred response immediately

	var userBalance float64
	var userID string

	// Safely get user ID
	if i.Member != nil && i.Member.User != nil {
		userID = i.Member.User.ID
	} else if i.User != nil {
		userID = i.User.ID
	} else {
		log.Printf("Warning: Could not determine user ID for interaction in guild %s", i.GuildID)
		return
	}

	if banChk(s, i, userID) {
		return
	}

	// Use context with timeout for DB operations
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	var gameType sql.NullString
	err := db.QueryRowContext(ctx, "SELECT type FROM active_games WHERE userid = ? AND type = ? LIMIT 1", userID, "slot").Scan(&gameType)
	if err == sql.ErrNoRows {
		// No active game
		if err := saveActiveslotToDB(userID); err != nil {
			log.Println("Error saving game:", err)
		}
		// return
	} else if err != nil {
		log.Println("query error:", err)
		return
	}

	if gameType.Valid && gameType.String == "slot" {
		respondEphemeral(s, i, "❌ You already have an active slot game", nil)
		return
	}

	err = db.QueryRowContext(ctx, "SELECT balance FROM users WHERE userid = ?", userID).Scan(&userBalance)
	if err == sql.ErrNoRows {
		respondEphemeral(s, i, "❌ You're not registered! Registering user...", nil)
		addUser(s, i)
		return
	} else if err != nil {
		log.Printf("DB error checking balance for user %s: %v", userID, err)
		respondEphemeral(s, i, "❌ Database error!", nil)
		return
	}

	// Validation checks
	if userBalance < betAmount {
		respondEphemeral(s, i, "❌ Insufficient balance!", nil)
		return
	}
	if betAmount <= 0 {
		respondEphemeral(s, i, "❌ Invalid amount!", nil)
		return
	}
	err = s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseDeferredChannelMessageWithSource,
	})
	if err != nil {
		log.Printf("failed to defer response: %v", err)
		return
	}
	// Generate results (moved earlier to reduce time between DB operations)
	result := make([]int, 3)
	for j := 0; j < 3; j++ {
		result[j] = rand.Intn(12)
	}

	// Calculate payout
	counts := make(map[int]int)
	for _, v := range result {
		counts[v]++
	}

	var payout float64 = 0
	for sym, c := range counts {
		if sym == 7 { // skip ❌
			continue
		}
		if c == 3 {
			if sym == 0 { // 7️⃣ jackpot
				payout = 77.7
			} else {
				payout = 33.3
			}
			break
		} else if c == 2 {
			if sym == 0 { // two 7️⃣
				payout = 7.7
			} else {
				payout = 3
			}
			break
		}
	}

	// Calculate win amount before transaction
	var winAmount float64
	if payout > 0 {
		winAmount = payout * betAmount
	} else {
		winAmount = -betAmount
	}

	// Start transaction with context
	tx, err := db.BeginTx(ctx, &sql.TxOptions{Isolation: sql.LevelReadCommitted})
	if err != nil {
		log.Printf("DB error starting transaction for user %s: %v", userID, err)
		respondEphemeral(s, i, "❌ Database error!", nil)
		return
	}
	defer tx.Rollback()
	go logGame(userID, "slot", betAmount, winAmount)

	// Use a single UPDATE query to handle balance, wins, and losses
	var query string
	if payout > 0 {
		query = `UPDATE users 
		         SET balance = ROUND(balance + ?, 2),
		             wins = ROUND(wins + ?, 2)
		         WHERE userid = ? AND balance >= ?`
	} else {
		query = `UPDATE users 
		         SET balance = ROUND(balance + ?, 2),
		             losses = ROUND(losses + ?, 2)
		         WHERE userid = ? AND balance >= ?`
	}

	result_exec, err := tx.ExecContext(ctx, query, winAmount, math.Abs(winAmount), userID, betAmount)
	if err != nil {
		log.Printf("DB error updating user %s: %v", userID, err)
		respondEphemeral(s, i, "❌ Database error!", nil)
		return
	}

	// Check if the update affected any rows (handles race condition)
	rowsAffected, err := result_exec.RowsAffected()
	if err != nil || rowsAffected == 0 {
		log.Printf("Race condition detected for user %s - insufficient balance", userID)
		respondEphemeral(s, i, "❌ Insufficient balance or concurrent transaction!", nil)
		return
	}
	// Commit transaction
	if err := tx.Commit(); err != nil {
		log.Printf("DB error committing transaction for user %s: %v", userID, err)
		respondEphemeral(s, i, "❌ Database error!", nil)
		return
	}

	// Update final balance
	userBalance += winAmount

	color := 0xFFFF00 // yellow

	username := fmt.Sprintf("<@%s>", userID)

	// Create disabled button
	playAgainButton := &discordgo.Button{
		Label:    "Play Again 🎰",
		Style:    discordgo.PrimaryButton,
		CustomID: fmt.Sprintf("playagainSlot_%v", betAmount),
		Disabled: true,
	}
	row := discordgo.ActionsRow{
		Components: []discordgo.MessageComponent{playAgainButton},
	}

	// Animation sequence (non-blocking)
	editWithRetry := func(edit *discordgo.WebhookEdit) {
		for attempt := 0; attempt < 3; attempt++ {
			_, err := s.InteractionResponseEdit(i.Interaction, edit)
			if err == nil {
				return
			}
			if attempt < 2 {
				time.Sleep(100 * time.Millisecond)
			} else {
				log.Printf("Failed to edit interaction after 3 attempts: %v", err)
			}
		}
	}

	// Step 1: Initial loading
	content := fmt.Sprintf("> **%s's Game**", username)
	editWithRetry(&discordgo.WebhookEdit{
		Content: &content,
		Embeds: &[]*discordgo.MessageEmbed{
			buildEmbed(buildEmojiGrid("loading", "loading", "loading", s), color, betAmount, nil, userBalance-winAmount),
		},
		Components: &[]discordgo.MessageComponent{row},
	})

	// Step 2: First slot
	time.Sleep(time.Duration(rand.Intn(1000)+500) * time.Millisecond)

	editWithRetry(&discordgo.WebhookEdit{
		Embeds: &[]*discordgo.MessageEmbed{
			buildEmbed(buildEmojiGrid(strconv.Itoa(result[0]), "loading", "loading", s), color, betAmount, nil, userBalance-winAmount),
		},
		Components: &[]discordgo.MessageComponent{row},
	})

	// Step 3: Second slot
	time.Sleep(time.Duration(rand.Intn(500)+250) * time.Millisecond)

	editWithRetry(&discordgo.WebhookEdit{
		Embeds: &[]*discordgo.MessageEmbed{
			buildEmbed(buildEmojiGrid(fmt.Sprintf("%dF", result[0]), strconv.Itoa(result[1]), "loading", s), color, betAmount, nil, userBalance-winAmount),
		},
		Components: &[]discordgo.MessageComponent{row},
	})

	// Step 4: Final result with enabled button
	if payout > 0 {
		color = 0x00FF00 // green
	} else {
		color = 0xFF0000 // red
	}

	enabledButton := &discordgo.Button{
		Label:    "Play Again 🎰",
		Style:    discordgo.PrimaryButton,
		CustomID: fmt.Sprintf("playagainSlot_%v", betAmount),
		Disabled: false,
	}
	enabledRow := discordgo.ActionsRow{
		Components: []discordgo.MessageComponent{enabledButton},
	}

	time.Sleep(time.Duration(rand.Intn(1000)+500) * time.Millisecond)
	editWithRetry(&discordgo.WebhookEdit{
		Embeds: &[]*discordgo.MessageEmbed{
			buildEmbed(buildEmojiGrid(fmt.Sprintf("%dF", result[0]), fmt.Sprintf("%dF", result[1]), strconv.Itoa(result[2]), s), color, betAmount, &payout, userBalance),
		},
		Components: &[]discordgo.MessageComponent{enabledRow},
	})

	// Step 5: Final state
	time.Sleep(100 * time.Millisecond)
	editWithRetry(&discordgo.WebhookEdit{
		Embeds: &[]*discordgo.MessageEmbed{
			buildEmbed(buildEmojiGrid(fmt.Sprintf("%dF", result[0]), fmt.Sprintf("%dF", result[1]), fmt.Sprintf("%dF", result[2]), s), color, betAmount, &payout, userBalance),
		},
		Components: &[]discordgo.MessageComponent{enabledRow},
	})
	// Delete active game
	if err := deleteActiveGameFromDB(userID, "slot"); err != nil {
		log.Printf("DB error deleting game for user %s: %v", userID, err)
		return
	}

}
