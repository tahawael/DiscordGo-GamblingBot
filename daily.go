package main

import (
	"database/sql"
	"fmt"
	"math"
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

var (
	Min float64 // set to true for production (global commands)
	Max float64 // set to true for production (global commands)
)

// getRewardInfo returns the min, max, and a random actual reward for a streak.
func getRewardInfo(streak int) (reward float64) {

	if streak < 1 {
		streak = 1
	}

	// Base reward grows slowly with streak
	base := 1000.0 * math.Pow(float64(streak), 1.5) // exponential growth with diminishing returns
	spread := base * 0.5                            // random spread ±50%

	Min = math.Max(base-spread, 500) // minimum reward floor
	Max = base + spread

	reward = Min + rand.Float64()*(Max-Min)
	return
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
		}
	} else if err != sql.ErrNoRows {
		return nil, err
	}

	// Calculate reward
	award := getRewardInfo(streak)

	// Insert reward claim
	_, err = tx.Exec(`
		INSERT INTO daily_rewards (userid, claim_date, streak, reward_amount)
		VALUES (?, ?, ?, ?)
	`, userID, today, streak, award)
	if err != nil {
		return nil, err
	}

	// Update user balance
	_, err = tx.Exec(`
		UPDATE users 
		SET balance = balance + ? 
		WHERE userid = ?
	`, award, userID)
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
		RewardAmount: award,
	}, nil
}

// GetCurrentStreak returns the user's current streak based on last claim
func GetCurrentStreak(db *sql.DB, userID string) (int, error) {
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
		return 0, nil // No streak yet
	}
	if err != nil {
		return 0, err
	}

	yesterday := time.Now().AddDate(0, 0, -1).Format("2006-01-02")
	today := time.Now().Format("2006-01-02")

	// Continue streak if last claim was yesterday
	if lastClaimDate == yesterday {
		return streak, nil
	}
	// If claimed today, streak stays the same
	if lastClaimDate == today {
		return streak, nil
	}

	// Missed a day — reset streak
	return 0, nil
}

// HandleDailyCommand handles the /daily command
func HandleDailyCommand(s *discordgo.Session, i *discordgo.InteractionCreate, db *sql.DB, userID string) {
	currentStreak, _ := GetCurrentStreak(db, userID)

	today := time.Now().Format("2006-01-02")
	var exists int
	err := db.QueryRow(`
	SELECT 1 FROM daily_rewards
	WHERE userid = ? AND claim_date = ?
	`, userID, today).Scan(&exists)

	alreadyClaimed := (err == nil)

	// Determine next claimable day
	nextDay := currentStreak + 1
	if currentStreak == 0 {
		nextDay = 1
	}

	msg := "**Daily Rewards**\nClaim your rewards each day to build your streak!"

	var style discordgo.ButtonStyle
	var disabled bool
	var emoji string

	if alreadyClaimed {
		style = discordgo.SuccessButton
		disabled = true
		emoji = "🔒"
	} else {
		style = discordgo.PrimaryButton
		disabled = false
		emoji = "🎁"
	}

	getRewardInfo(nextDay)
	btn := discordgo.Button{
		Label: fmt.Sprintf("Streak %d\n$%.2f-$%.2f", nextDay, Min, Max),

		Style:    style,
		CustomID: fmt.Sprintf("daily_claim_%d", nextDay),
		Disabled: disabled,
		Emoji: &discordgo.ComponentEmoji{
			Name: emoji,
		},
	}

	row := discordgo.ActionsRow{
		Components: []discordgo.MessageComponent{btn},
	}

	respondEphemeral(s, i, msg, []discordgo.MessageComponent{row})
}

// HandleDailyClaimButton handles when user clicks the "Claim" button
func HandleDailyClaimButton(s *discordgo.Session, i *discordgo.InteractionCreate, db *sql.DB, userID string) {
	reward, err := ClaimDailyReward(db, userID)
	if err != nil {
		if err.Error() == "already claimed today" {
			nextClaim := time.Date(
				time.Now().Year(),
				time.Now().Month(),
				time.Now().Day()+1,
				0, 0, 0, 0,
				time.Now().Location(),
			)

			timeUntil := time.Until(nextClaim)
			hours := int(timeUntil.Hours())
			minutes := int(timeUntil.Minutes()) % 60

			msg := fmt.Sprintf(
				"⏰ **Already Claimed!**\n\nYou've already claimed your daily reward today.\nCome back in **%dh %dm** to claim again!",
				hours, minutes,
			)

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
	msg := fmt.Sprintf(
		"✅ **Daily Reward Claimed!**\n\n🔥 Streak: **Day %d**\n💰 Reward: **$%.2f**\n\nCome back tomorrow to continue your streak!",
		reward.Streak,
		reward.RewardAmount,
	)

	btn := discordgo.Button{
		Label:    fmt.Sprintf("Streak %d", reward.Streak),
		Style:    discordgo.SuccessButton,
		CustomID: fmt.Sprintf("daily_claim_%d", reward.Streak),
		Disabled: true,
		Emoji: &discordgo.ComponentEmoji{
			Name: "✅",
		},
	}

	row := discordgo.ActionsRow{
		Components: []discordgo.MessageComponent{btn},
	}

	// Update original message
	s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseUpdateMessage,
		Data: &discordgo.InteractionResponseData{
			Content:    "🎁 **Daily Rewards**\n\nClaim your rewards each day to build your streak!",
			Components: []discordgo.MessageComponent{row},
		},
	})

	// Follow-up confirmation
	s.FollowupMessageCreate(i.Interaction, true, &discordgo.WebhookParams{
		Content: msg,
		Flags:   discordgo.MessageFlagsEphemeral,
	})
}
