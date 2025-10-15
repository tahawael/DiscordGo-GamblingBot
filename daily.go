package main

import (
	"database/sql"
	"fmt"
	"math/rand"
	"time"

	"github.com/bwmarrin/discordgo"
)

type DailyReward struct {
	UserID       string
	ClaimDate    time.Time
	Streak       int
	RewardAmount float64
}

// getRewardAmount returns a random reward amount based on streak
func getRewardAmount(streak int) float64 {
	switch streak {
	case 1:
		return float64(rand.Intn(1000) + 1000) // 1k-2k
	case 2:
		return float64(rand.Intn(1000) + 2000) // 2k-3k
	case 3:
		return float64(rand.Intn(1000) + 3000) // 3k-4k
	case 4:
		return float64(rand.Intn(1000) + 4000) // 4k-5k
	case 5:
		return float64(rand.Intn(5000) + 5000) // 5k-10k
	case 6:
		return float64(rand.Intn(40000) + 10000) // 10k-50k
	case 7:
		return float64(rand.Intn(50000) + 50000) // 50k-100k
	default:
		return float64(rand.Intn(1000) + 1000)
	}
}

// ClaimDailyReward processes a daily reward claim
func ClaimDailyReward(db *sql.DB, userID string) (*DailyReward, error) {
	tx, err := db.Begin()
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()

	today := time.Now().Format("2006-01-02")
	yesterday := time.Now().AddDate(0, 0, -1).Format("2006-01-02")

	// Check if already claimed today
	var exists int
	err = tx.QueryRow("SELECT 1 FROM daily_rewards WHERE userid = ? AND claim_date = ?", userID, today).Scan(&exists)
	if err == nil {
		return nil, fmt.Errorf("already claimed today")
	} else if err != sql.ErrNoRows {
		return nil, err
	}

	// Get last claim to calculate streak
	var lastStreak int
	var lastClaimDate string
	err = tx.QueryRow(`
		SELECT streak, claim_date 
		FROM daily_rewards 
		WHERE userid = ? 
		ORDER BY claimed_at DESC 
		LIMIT 1
	`, userID).Scan(&lastStreak, &lastClaimDate)

	streak := 1
	if err == nil {
		if lastClaimDate == yesterday {
			streak = lastStreak + 1
			if streak > 7 {
				streak = 1
			}
		}
	} else if err != sql.ErrNoRows {
		return nil, err
	}

	// Calculate reward
	rewardAmount := getRewardAmount(streak)

	// Insert reward claim
	_, err = tx.Exec(`
		INSERT INTO daily_rewards (userid, claim_date, streak, reward_amount)
		VALUES (?, ?, ?, ?)
	`, userID, today, streak, rewardAmount)
	if err != nil {
		return nil, err
	}

	// Update user balance
	_, err = tx.Exec(`
		UPDATE users 
		SET balance = balance + ? 
		WHERE userid = ?
	`, rewardAmount, userID)
	if err != nil {
		return nil, err
	}

	if err = tx.Commit(); err != nil {
		return nil, err
	}

	return &DailyReward{
		UserID:       userID,
		ClaimDate:    time.Now(),
		Streak:       streak,
		RewardAmount: rewardAmount,
	}, nil
}

// GetCurrentStreak returns the user's current streak
func GetCurrentStreak(db *sql.DB, userID string) (int, error) {
	yesterday := time.Now().AddDate(0, 0, -1).Format("2006-01-02")
	today := time.Now().Format("2006-01-02")

	var streak int
	var lastClaimDate string
	err := db.QueryRow(`
		SELECT streak, claim_date 
		FROM daily_rewards 
		WHERE userid = ? 
		ORDER BY claimed_at DESC 
		LIMIT 1
	`, userID).Scan(&streak, &lastClaimDate)

	if err == sql.ErrNoRows {
		return 0, nil
	}
	if err != nil {
		return 0, err
	}

	// Streak is valid if claimed today or yesterday
	if lastClaimDate == today || lastClaimDate == yesterday {
		return streak, nil
	}

	return 0, nil // Streak broken
}

// HandleDailyCommand handles the /daily slash command
func HandleDailyCommand(s *discordgo.Session, i *discordgo.InteractionCreate, db *sql.DB, userID string) {
	// userID := parseUserID(i.Member.User.ID)
	currentStreak, _ := GetCurrentStreak(db, userID)

	// Check if already claimed today
	today := time.Now().Format("2006-01-02")
	var exists int
	err := db.QueryRow("SELECT 1 FROM daily_rewards WHERE userid = ? AND claim_date = ?", userID, today).Scan(&exists)
	alreadyClaimed := (err == nil)

	var nextDay int
	if alreadyClaimed {
		// User already claimed today, next claimable day is tomorrow
		nextDay = currentStreak + 1
	} else {
		// User has not claimed today
		if currentStreak == 0 {
			nextDay = 1
		} else {
			nextDay = currentStreak + 1
		}
	}

	if nextDay > 7 {
		nextDay = 1
	}

	msg := "🎁 **Daily Rewards**\n\nClaim your rewards each day to build your streak!"

	// Reward info for each day
	rewards := []string{
		"($1K-2K)",
		"($2K-3K)",
		"($3K-4K)",
		"($4K-5K)",
		"($5K-10K)",
		"($10K-50K)",
		"($50K-100K)",
	}

	// Create 7 buttons
	row1 := []discordgo.MessageComponent{}
	row2 := []discordgo.MessageComponent{}

	for day := 1; day <= 7; day++ {
		var style discordgo.ButtonStyle
		var disabled bool
		var emoji string

		if day < nextDay || (alreadyClaimed && day == currentStreak) {
			// Already claimed
			style = discordgo.SuccessButton
			disabled = true
			emoji = "✅"
		} else if day == nextDay && !alreadyClaimed {
			// Claimable today
			style = discordgo.SuccessButton
			disabled = false
			emoji = "🎁"
		} else {
			// Future days
			style = discordgo.SecondaryButton
			disabled = true
			emoji = "🔒"
		}

		btn := discordgo.Button{
			Label:    fmt.Sprintf("Day %d\n%s", day, rewards[day-1]),
			Style:    style,
			CustomID: fmt.Sprintf("daily_claim_%d", day),
			Disabled: disabled,
			Emoji: &discordgo.ComponentEmoji{
				Name: emoji,
			},
		}

		if day <= 4 {
			row1 = append(row1, btn)
		} else {
			row2 = append(row2, btn)
		}
	}

	components := []discordgo.MessageComponent{
		discordgo.ActionsRow{Components: row1},
		discordgo.ActionsRow{Components: row2},
	}

	respondEphemeral(s, i, msg, components)
}

// HandleDailyClaimButton handles the claim button interaction
func HandleDailyClaimButton(s *discordgo.Session, i *discordgo.InteractionCreate, db *sql.DB, userID string) {

	reward, err := ClaimDailyReward(db, userID)
	if err != nil {
		if err.Error() == "already claimed today" {
			nextClaim := time.Now().AddDate(0, 0, 1)
			nextClaim = time.Date(nextClaim.Year(), nextClaim.Month(), nextClaim.Day(), 0, 0, 0, 0, nextClaim.Location())
			timeUntil := time.Until(nextClaim)
			hours := int(timeUntil.Hours())
			minutes := int(timeUntil.Minutes()) % 60

			msg := fmt.Sprintf("⏰ **Already Claimed!**\n\n"+
				"You've already claimed your daily reward today.\n"+
				"Come back in **%dh %dm** to claim again!", hours, minutes)

			s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
				Type: discordgo.InteractionResponseChannelMessageWithSource,
				Data: &discordgo.InteractionResponseData{
					Content: msg,
					Flags:   discordgo.MessageFlagsEphemeral,
				},
			})
			return
		}

		s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseChannelMessageWithSource,
			Data: &discordgo.InteractionResponseData{
				Content: "❌ Failed to claim reward. Please try again.",
				Flags:   discordgo.MessageFlagsEphemeral,
			},
		})
		return
	}

	// Success message
	msg := fmt.Sprintf("✅ **Daily Reward Claimed!**\n\n"+
		"🔥 Streak: **Day %d**\n"+
		"💰 Reward: **$%.2f**\n\n"+
		"Come back tomorrow to continue your streak!", reward.Streak, reward.RewardAmount)

	if reward.Streak == 7 {
		msg += "\n\n🎉 **Congratulations!** You've completed the 7-day cycle!"
	}

	// Update the original message with disabled buttons
	currentStreak := reward.Streak
	rewards := []string{
		"1K-2K",
		"2K-3K",
		"3K-4K",
		"4K-5K",
		"5K-10K",
		"10K-50K",
		"50K-100K",
	}

	row1 := []discordgo.MessageComponent{}
	row2 := []discordgo.MessageComponent{}

	for day := 1; day <= 7; day++ {
		var style discordgo.ButtonStyle
		var emoji string

		if day <= currentStreak {
			style = discordgo.PrimaryButton
			emoji = "✅"
		} else {
			style = discordgo.SecondaryButton
			emoji = "🔒"
		}

		btn := discordgo.Button{
			Label:    fmt.Sprintf("Day %d\n%s", day, rewards[day-1]),
			Style:    style,
			CustomID: fmt.Sprintf("daily_claim_%d", day),
			Disabled: true,
			Emoji: &discordgo.ComponentEmoji{
				Name: emoji,
			},
		}

		if day <= 4 {
			row1 = append(row1, btn)
		} else {
			row2 = append(row2, btn)
		}
	}

	components := []discordgo.MessageComponent{
		discordgo.ActionsRow{Components: row1},
		discordgo.ActionsRow{Components: row2},
	}

	// Update original message
	s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseUpdateMessage,
		Data: &discordgo.InteractionResponseData{
			Content:    "🎁 **Daily Rewards**\n\nClaim your rewards each day to build your streak!",
			Components: components,
		},
	})

	// Send success message as follow-up
	s.FollowupMessageCreate(i.Interaction, true, &discordgo.WebhookParams{
		Content: msg,
		Flags:   discordgo.MessageFlagsEphemeral,
	})
}
