package main

import (
	"database/sql"
	"fmt"
	"log"

	"github.com/bwmarrin/discordgo"
)

// respondEphemeral sends an ephemeral response (works even if already responded).
func respondEphemeral(s *discordgo.Session, i *discordgo.InteractionCreate, msg string, components []discordgo.MessageComponent) error {
	err := s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{
			Content:    msg,
			Flags:      discordgo.MessageFlagsEphemeral,
			Components: components, // ✅ correct field
		},
	})

	if err != nil {
		// If already responded, fall back to follow-up
		_, followupErr := s.FollowupMessageCreate(i.Interaction, true, &discordgo.WebhookParams{
			Content:    msg,
			Flags:      discordgo.MessageFlagsEphemeral,
			Components: components,
		})
		if followupErr != nil {
			return followupErr
		}
	}
	return nil
}

func respondUpdate(s *discordgo.Session, i *discordgo.InteractionCreate, content string, components []discordgo.MessageComponent, game *MinesGame) error {
	newComponents := components
	stringPtr := func(s string) *string { return &s }

	game.mu.Lock()
	deferred := game.deferred
	game.mu.Unlock()

	// log.Println("game.deferred is:", deferred)

	if !deferred {
		// Try immediate update first
		err := s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseUpdateMessage,
			Data: &discordgo.InteractionResponseData{
				Content:    content,
				Components: newComponents,
			},
		})

		if err != nil {
			log.Println("InteractionRespond failed, falling back to edit:", err)

			// Fallback to deferred edit
			_, editErr := s.InteractionResponseEdit(i.Interaction, &discordgo.WebhookEdit{
				Content:    stringPtr(content),
				Components: &newComponents,
			})
			if editErr != nil {
				log.Println("InteractionResponseEdit also failed:", editErr)
				return editErr
			}
		}
	} else {
		// Use deferred edit
		_, err := s.InteractionResponseEdit(i.Interaction, &discordgo.WebhookEdit{
			Content:    stringPtr(content),
			Components: &newComponents,
		})
		if err != nil {
			log.Println("InteractionResponseEdit error:", err)
			return err
		}
	}

	return nil
}

func sendNewMessage(s *discordgo.Session, i *discordgo.InteractionCreate, content string, components []discordgo.MessageComponent) error {
	// Attempt to respond to the interaction
	err := s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{
			Content:    content,
			Components: components,
		},
	})

	if err != nil {
		// If already responded (Discord returns 40060 or other), send follow-up instead
		_, followupErr := s.FollowupMessageCreate(i.Interaction, true, &discordgo.WebhookParams{
			Content:    content,
			Components: components,
		})
		return followupErr
	}
	return nil
}

func banChk(s *discordgo.Session, i *discordgo.InteractionCreate, userID string) bool {
	var isBanned bool
	err := db.QueryRow("SELECT banned FROM users WHERE userid = ?", userID).Scan(&isBanned)
	if err != nil {
		if err == sql.ErrNoRows {
			fmt.Println("User not found")
			return false
		}
		log.Println("Query error:", err)
		return false
	}

	if isBanned {
		if err := respondEphemeral(s, i, "❌ You're Banned.", nil); err != nil {
			log.Println("respondUpdate error (is Banned):", err)
		}
		return true
	}

	return false
}

// adminChk checks if the caller is an admin.
func adminChk(s *discordgo.Session, i *discordgo.InteractionCreate, userID string) bool {
	var isAdmin int

	err := db.QueryRow("SELECT admin FROM users WHERE userid = ?", userID).Scan(&isAdmin)
	if err != nil {
		if err == sql.ErrNoRows {
			if err := respondEphemeral(s, i, "❌ You're not registered yet.", nil); err != nil {
				log.Println("respondUpdate error (not registered):", err)
			}
			return false
		}
		log.Println("DB error (admin check):", err)
		if err := respondEphemeral(s, i, "⚠️ Database error, please try again later.", nil); err != nil {
			log.Println("respondUpdate error (db error):", err)
		}
		return false
	}

	if isAdmin == 0 {
		if err := respondEphemeral(s, i, "❌ You're not an admin.", nil); err != nil {
			log.Println("respondUpdate error (not admin):", err)
		}
		return false
	}

	return true
}
