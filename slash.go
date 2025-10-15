package main

import (
	"database/sql"
	"fmt"
	"log"
	"strconv"
	"strings"

	"github.com/bwmarrin/discordgo"
)

var (
	//local servers for rapid prototyping and
	GuildIDs = []string{
		"", // Server 1
		"",  // Server 2...
	}

	isGlb = true // set to true for production (global commands)
)

var commands = []*discordgo.ApplicationCommand{

	{
		Name:        "preload",
		Description: "if you're having problems with the animations",
		// Type:        discordgo.ChatApplicationCommand,
		IntegrationTypes: &[]discordgo.ApplicationIntegrationType{
			discordgo.ApplicationIntegrationGuildInstall,
			discordgo.ApplicationIntegrationUserInstall,
		},
		Contexts: &[]discordgo.InteractionContextType{
			discordgo.InteractionContextGuild,
			discordgo.InteractionContextBotDM,
			discordgo.InteractionContextPrivateChannel,
		},
	},
	{
		Name:        "daily",
		Description: "Claim your daily reward",
		// Type:        discordgo.ChatApplicationCommand,
		IntegrationTypes: &[]discordgo.ApplicationIntegrationType{
			discordgo.ApplicationIntegrationGuildInstall,
			discordgo.ApplicationIntegrationUserInstall,
		},
		Contexts: &[]discordgo.InteractionContextType{
			discordgo.InteractionContextGuild,
			discordgo.InteractionContextBotDM,
			discordgo.InteractionContextPrivateChannel,
		},
	},

	{
		Name:        "check-balance",
		Description: "check your or balance someone else's",
		// Type:        discordgo.ChatApplicationCommand,
		IntegrationTypes: &[]discordgo.ApplicationIntegrationType{
			discordgo.ApplicationIntegrationGuildInstall,
			discordgo.ApplicationIntegrationUserInstall,
		},
		Contexts: &[]discordgo.InteractionContextType{
			discordgo.InteractionContextGuild,
			discordgo.InteractionContextBotDM,
			discordgo.InteractionContextPrivateChannel,
		},

		Options: []*discordgo.ApplicationCommandOption{
			{
				Type:        discordgo.ApplicationCommandOptionUser,
				Name:        "user",
				Description: "who do you wanna check",
				Required:    false,
			},
		},
	},
	{
		Name:        "transfer-balance",
		Description: "Transfer balance to a user",
		// Type:        discordgo.ChatApplicationCommand,
		IntegrationTypes: &[]discordgo.ApplicationIntegrationType{
			discordgo.ApplicationIntegrationGuildInstall,
			discordgo.ApplicationIntegrationUserInstall,
		},
		Contexts: &[]discordgo.InteractionContextType{
			discordgo.InteractionContextGuild,
			discordgo.InteractionContextBotDM,
			discordgo.InteractionContextPrivateChannel,
		},

		Options: []*discordgo.ApplicationCommandOption{
			{
				Type:        discordgo.ApplicationCommandOptionUser,
				Name:        "user",
				Description: "who do you wanna transfer to",
				Required:    true,
			},
			{
				Type:        discordgo.ApplicationCommandOptionNumber,
				Name:        "amount",
				Description: "the amount you wanna transfer",
				Required:    true,
			},
		},
	},
	{
		Name:        "mines",
		Description: "Play Mines game",
		// Type:        discordgo.ChatApplicationCommand,
		IntegrationTypes: &[]discordgo.ApplicationIntegrationType{
			discordgo.ApplicationIntegrationGuildInstall,
			discordgo.ApplicationIntegrationUserInstall,
		},
		Contexts: &[]discordgo.InteractionContextType{
			discordgo.InteractionContextGuild,
			discordgo.InteractionContextBotDM,
			discordgo.InteractionContextPrivateChannel,
		},

		Options: []*discordgo.ApplicationCommandOption{
			{
				Type:        discordgo.ApplicationCommandOptionNumber,
				Name:        "bet_amount",
				Description: "Amount to bet",
				Required:    true,
			},
			{
				Type:        discordgo.ApplicationCommandOptionInteger,
				Name:        "number_of_mines",
				Description: "Number of mines on the board (1-15)",
				Required:    true,
			},
		},
	},
	{
		Name:        "slot",
		Description: "Play classic slot machine, use /preload if having issues with animations",
		// Type:        discordgo.ChatApplicationCommand,
		IntegrationTypes: &[]discordgo.ApplicationIntegrationType{
			discordgo.ApplicationIntegrationGuildInstall,
			discordgo.ApplicationIntegrationUserInstall,
		},
		Contexts: &[]discordgo.InteractionContextType{
			discordgo.InteractionContextGuild,
			discordgo.InteractionContextBotDM,
			discordgo.InteractionContextPrivateChannel,
		},

		Options: []*discordgo.ApplicationCommandOption{
			{
				Type:        discordgo.ApplicationCommandOptionNumber,
				Name:        "bet_amount",
				Description: "Amount to bet",
				Required:    true,
			},
		},
	},
}

// Helper function to compare options
func compareOptions(a, b []*discordgo.ApplicationCommandOption) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i].Name != b[i].Name || a[i].Description != b[i].Description || a[i].Type != b[i].Type || a[i].Required != b[i].Required {
			return false
		}
	}
	return true
}

func addCommands(dg *discordgo.Session) {
	// First, clean up old commands when switching modes
	if isGlb {
		// When switching to global, remove all guild-specific commands
		for _, gID := range GuildIDs {
			existing, err := dg.ApplicationCommands(dg.State.User.ID, gID)
			if err != nil {
				fmt.Printf("[ERROR] Failed to fetch guild commands for cleanup: %v\n", err)
				continue
			}

			for _, cmd := range existing {
				err := dg.ApplicationCommandDelete(dg.State.User.ID, gID, cmd.ID)
				if err != nil {
					fmt.Printf("[ERROR] Failed to delete old guild command '%s': %v\n", cmd.Name, err)
				} else {
					fmt.Printf("[CLEANUP] Deleted old guild command '%s' from guild %s\n", cmd.Name, gID)
				}
			}
		}

		log.Printf("[DEBUG] Registering commands globally...")

		existing, err := dg.ApplicationCommands(dg.State.User.ID, "")
		if err != nil {
			fmt.Printf("[ERROR] Failed to fetch global commands: %v\n", err)
			return
		}
		fmt.Printf("[DEBUG] Found %d existing global commands\n", len(existing))

		// Create a map for easier lookup
		desiredMap := make(map[string]*discordgo.ApplicationCommand)
		for _, cmd := range commands {
			desiredMap[cmd.Name] = cmd
		}

		// Update or create commands
		for _, cmd := range commands {
			var existingCmd *discordgo.ApplicationCommand
			for _, e := range existing {
				if e.Name == cmd.Name {
					existingCmd = e
					break
				}
			}

			if existingCmd != nil {
				if existingCmd.Description != cmd.Description || !compareOptions(existingCmd.Options, cmd.Options) {
					_, err := dg.ApplicationCommandEdit(dg.State.User.ID, "", existingCmd.ID, cmd)
					if err != nil {
						fmt.Printf("[ERROR] Failed to update global command '%s': %v\n", cmd.Name, err)
					} else {
						fmt.Printf("[UPDATE] Updated global command '%s'\n", cmd.Name)
					}
				} else {
					fmt.Printf("[SKIP] Global command '%s' is up-to-date\n", cmd.Name)
				}
			} else {
				newCmd, err := dg.ApplicationCommandCreate(dg.State.User.ID, "", cmd)
				if err != nil {
					fmt.Printf("[ERROR] Failed to create global command '%s': %v\n", cmd.Name, err)
				} else {
					fmt.Printf("[CREATE] Created global command '%s' with ID %s\n", cmd.Name, newCmd.ID)
				}
			}
		}

		// Delete obsolete global commands
		for _, e := range existing {
			if _, ok := desiredMap[e.Name]; !ok {
				err := dg.ApplicationCommandDelete(dg.State.User.ID, "", e.ID)
				if err != nil {
					fmt.Printf("[ERROR] Failed to delete obsolete global command '%s': %v\n", e.Name, err)
				} else {
					fmt.Printf("[DELETE] Deleted obsolete global command '%s'\n", e.Name)
				}
			}
		}

	} else {
		// When switching to guild mode, remove all global commands
		existing, err := dg.ApplicationCommands(dg.State.User.ID, "")
		if err != nil {
			fmt.Printf("[ERROR] Failed to fetch global commands for cleanup: %v\n", err)
		} else {
			for _, cmd := range existing {
				err := dg.ApplicationCommandDelete(dg.State.User.ID, "", cmd.ID)
				if err != nil {
					fmt.Printf("[ERROR] Failed to delete old global command '%s': %v\n", cmd.Name, err)
				} else {
					fmt.Printf("[CLEANUP] Deleted old global command '%s'\n", cmd.Name)
				}
			}
		}

		for _, gID := range GuildIDs {
			fmt.Printf("[DEBUG] Registering commands for guild ID: %s\n", gID)

			existing, err := dg.ApplicationCommands(dg.State.User.ID, gID)
			if err != nil {
				fmt.Printf("[ERROR] Failed to fetch commands for guild %s: %v\n", gID, err)
				continue
			}
			fmt.Printf("[DEBUG] Found %d existing commands in guild %s\n", len(existing), gID)

			// Create a map for easier lookup
			desiredMap := make(map[string]*discordgo.ApplicationCommand)
			for _, cmd := range commands {
				desiredMap[cmd.Name] = cmd
			}

			// Update or create commands
			for _, cmd := range commands {
				var existingCmd *discordgo.ApplicationCommand
				for _, e := range existing {
					if e.Name == cmd.Name {
						existingCmd = e
						break
					}
				}

				if existingCmd != nil {
					// Detect changes: description or options
					if existingCmd.Description != cmd.Description || !compareOptions(existingCmd.Options, cmd.Options) {
						_, err := dg.ApplicationCommandEdit(dg.State.User.ID, gID, existingCmd.ID, cmd)
						if err != nil {
							fmt.Printf("[ERROR] Failed to update command '%s': %v\n", cmd.Name, err)
						} else {
							fmt.Printf("[UPDATE] Updated command '%s'\n", cmd.Name)
						}
					} else {
						fmt.Printf("[SKIP] Command '%s' is up-to-date\n", cmd.Name)
					}
				} else {
					newCmd, err := dg.ApplicationCommandCreate(dg.State.User.ID, gID, cmd)
					if err != nil {
						fmt.Printf("[ERROR] Failed to create command '%s': %v\n", cmd.Name, err)
					} else {
						fmt.Printf("[CREATE] Created command '%s' with ID %s\n", cmd.Name, newCmd.ID)
					}
				}
			}

			// Delete commands that exist on Discord but are not in desired commands
			for _, e := range existing {
				if _, ok := desiredMap[e.Name]; !ok {
					err := dg.ApplicationCommandDelete(dg.State.User.ID, gID, e.ID)
					if err != nil {
						fmt.Printf("[ERROR] Failed to delete obsolete command '%s': %v\n", e.Name, err)
					} else {
						fmt.Printf("[DELETE] Deleted obsolete command '%s'\n", e.Name)
					}
				}
			}
		}
	}
}
func addUser(s *discordgo.Session, i *discordgo.InteractionCreate) {
	var userID string
	var Username string

	// Safely get user ID for ban check
	if i.Member != nil && i.Member.User != nil {
		userID = i.Member.User.ID
		Username = i.Member.User.Username

	} else if i.User != nil {
		userID = i.User.ID
		Username = i.User.Username

	} else {
		// Log this case and return - shouldn't happen normally
		log.Printf("Warning: Could not determine user ID for interaction in guild %s", i.GuildID)
		return
	}
	res, err := db.Exec(
		`INSERT IGNORE INTO users(userid, username, balance, wins, losses, admin) VALUES(?, ?, ?, ?, ?, ?)`,
		userID, Username, 1000, 0, 0, 0,
	)

	if err != nil {
		// real DB error
		log.Println("DB insert error:", err)
		respondEphemeral(s, i, "⚠️ An error occurred while saving your data.", nil)
		// s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		// 	Type: discordgo.InteractionResponseChannelMessageWithSource,
		// 	Data: &discordgo.InteractionResponseData{
		// 		Content: "⚠️ An error occurred while saving your data.",
		// 	},
		// })
		return
	}

	rows, err := res.RowsAffected()
	if err != nil {
		// should rarely fail, but check anyway
		log.Println("RowsAffected error:", err)
	}

	var msg string
	if rows > 0 {
		msg = "✅ User added successfully!"
	} else {
		msg = "ℹ️ User already exists."
	}
	respondEphemeral(s, i, msg, nil)
}

// Handle slash command interactions
func interactionCreate(s *discordgo.Session, i *discordgo.InteractionCreate) {
	var userID string
	var msg string
	var balance float64
	var admin bool
	var username string

	// // Ignore direct messages
	// if i.GuildID == "" {
	// 	return
	// }

	// Only handle application command interactions (slash commands)
	if i.Type != discordgo.InteractionApplicationCommand {
		return
	}

	// Safely get user ID for ban check
	if i.Member != nil && i.Member.User != nil {
		userID = i.Member.User.ID
		username = i.Member.User.Username
	} else if i.User != nil {
		userID = i.User.ID
		username = i.User.Username
	} else {
		// Log this case and return - shouldn't happen normally
		log.Printf("Warning: Could not determine user ID for interaction in guild %s", i.GuildID)
		return
	}
	// log.Printf("%s used %v:", username, i.ApplicationCommandData())

	if banChk(s, i, userID) {
		return // stop here if banned
	}
	err := db.QueryRow(
		"SELECT balance, admin FROM users WHERE userid = ?",
		userID,
	).Scan(&balance, &admin)

	if err != nil {
		log.Println("DB error:", err)
		msg = "⚠️ Database error, please try again later."
	}
	if err == sql.ErrNoRows {
		msg = "❌ User is not registered yet, registring user..."
		respondEphemeral(s, i, msg, nil)
		addUser(s, i)
	}
	switch i.ApplicationCommandData().Name {
	case "slot":
		slot(s, i, i.ApplicationCommandData().Options[0].Value.(float64))

	case "mines":
		betAmount := i.ApplicationCommandData().Options[0].FloatValue()
		numMines := i.ApplicationCommandData().Options[1].IntValue()

		startMinesGame(s, i, userID, betAmount, numMines, balance)

	case "transfer-balance":
		var RxBlns float64
		var Rxusername string
		var msg string
		amount := i.ApplicationCommandData().Options[1].FloatValue()

		receiverID, _ := strconv.Atoi(strings.Trim(i.ApplicationCommandData().Options[0].Value.(string), "<@!>"))
		senderId, _ := strconv.Atoi(userID)
		err := db.QueryRow(
			"SELECT username, balance FROM users WHERE userid = ?",
			receiverID,
		).Scan(&Rxusername, &RxBlns)
		if err == sql.ErrNoRows {
			msg = "❌ Receiver is not registered."
		} else if senderId == receiverID && !admin {
			msg = "❌ You cannot transfer to yourself."
		} else if (amount > balance || amount <= 0) && !admin {
			msg = "❌ insufficient funds"
		} else if err != nil {
			log.Println("DB error:", err)
			msg = "⚠️ Database error, please try again later."
		} else {
			// Update balance in DB
			_, errUpdate := db.Exec(
				`UPDATE users
				SET balance = CASE
					WHEN userid = ? THEN balance - ?
					WHEN userid = ? THEN balance + ?
				END
				WHERE userid IN (?, ?)`,
				senderId, amount, receiverID, amount, senderId, receiverID,
			)
			if errUpdate != nil {
				log.Println("DB update error:", errUpdate)
				logTrans(senderId, username, receiverID, Rxusername, amount, errUpdate.Error())
				msg = fmt.Sprintf("⚠️ Failed to update balance: %s", errUpdate.Error())
			} else {
				// Get new balance
				err = db.QueryRow(
					"SELECT balance FROM users WHERE userid = ?",
					receiverID,
				).Scan(&RxBlns)
				if err != nil {
					log.Println("DB error after update:", err)
					msg = "⚠️ Error fetching updated balance."
				} else {
					msg = fmt.Sprintf("💰 %s Transferred %.2f, %s's balance is now %.2f", username, amount, Rxusername, RxBlns)
				}
				logTrans(senderId, username, receiverID, Rxusername, amount, "Success")
			}
		}
		sendNewMessage(s, i, msg, nil)

	case "check-balance":
		var msg, menName string
		var menBalance float64
		if len(i.ApplicationCommandData().Options) > 0 {
			// If a user is mentioned, get their ID
			opt := i.ApplicationCommandData().Options[0]
			menId := opt.UserValue(nil).ID // safer way to get the user ID
			err := db.QueryRow(
				"SELECT username, balance FROM users WHERE userid = ?",
				menId,
			).Scan(&menName, &menBalance)

			if err == sql.ErrNoRows {
				msg = "❌ Mentioned User is not registered yet."
			} else if err != nil {
				log.Println("DB error:", err)
				msg = "⚠️ Database error, please try again later."
			} else {
				msg = fmt.Sprintf("💰 %s's balance is %.2f", menName, menBalance)
			}
		} else {
			msg = fmt.Sprintf("💰 %s's balance is %.2f", username, balance)
		}
		s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseChannelMessageWithSource,
			Data: &discordgo.InteractionResponseData{
				Content: msg,
			},
		})

	case "preload":

		sendAnimatedEmojiGridBatched(s, i, 12, "Emoji Grids", 0x00ff00)
	case "daily":
		HandleDailyCommand(s, i, db, userID)

	}

}
