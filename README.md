
<h1 id="top" align="center">
  <a href="https://github.com/yourusername/DiscordGo-GamblingBot" target="_blank">
    <img src="https://i.ibb.co/Q3FqM6xv/Anime-Casino-Games-A-Fresh-Take-on-Online-Gambling-in-Japan-1.png" alt="DiscordGo GamblingBot" width="600">
  </a>
  <br>
  DiscordGo GamblingBot
  <br>
</h1>

<p align="center">
  <strong>A fun, modular, and fast gambling bot built with DiscordGo</strong>
</p>

<p align="center">
  <a href="https://discord.com/oauth2/authorize?client_id=1414462464862453791" target="_blank">
    <img src="https://img.shields.io/badge/Try%20Bot-5865F2?logo=discord&logoColor=white&style=flat" alt="Add to Discord">
  </a>
  <a href="https://github.com/bwmarrin/discordgo" target="_blank">
    <img src="https://img.shields.io/badge/discordgo-library-blue?style=flat" alt="DiscordGo">
  </a>
  <a href="https://golang.org/dl/" target="_blank">
    <img src="https://img.shields.io/badge/Go-1.18+-00ADD8?logo=go&logoColor=white&style=flat" alt="Go Version">
  </a>
  <a href="https://www.mysql.com/" target="_blank">
    <img src="https://img.shields.io/badge/MySQL-8.0+-4479A1?logo=mysql&logoColor=white&style=flat" alt="MySQL">
  </a>
</p>

<p align="center">
  <a href="#-features">Features</a> •
  <a href="#-quick-start">Quick Start</a> •
  <a href="#installation">Installation</a> •
  <a href="#%EF%B8%8F-configuration">Configuration</a>
</p>

---

## 📖 Overview

A feature-rich Discord gambling bot written in Go. Built with performance and modularity in mind, it features a complete economy system, multiple casino games, persistent player statistics, and a web dashboard for monitoring.

## ✨ Features

### 🎮 Core Functionality
- **Dual Mode Support**: Works as both a server-wide bot and individual user application
- **Complete Economy System**: Full balance management with earnings, transfers, and transaction history
- **Multiple Casino Games**: Mines, Slots, and more coming soon
- **Persistent Statistics**: Comprehensive tracking of wins, losses, and gameplay history
- **Daily Rewards System**: Claim daily rewards with an engaging streak multiplier system

### 🛡️ Administration & Security
- **Admin Controls**: Comprehensive moderation tools including user banning
- **Transaction Logging**: Complete audit trail of all financial activities
- **Database-Backed**: Reliable MySQL persistence for all user data

### 📊 Web Dashboard
- **Real-time Statistics**: Monitor bot performance and usage
- **User Analytics**: View player counts, game statistics, and economy health
- **Access via**: `http://localhost:8080` (configurable)

---

## 🚀 Quick Start

### Prerequisites

Before you begin, ensure you have the following installed:

- **Go** (version 1.18 or higher) - [Download here](https://golang.org/dl/)
- **MySQL Server** (version 8.0 or higher) - [Download here](https://dev.mysql.com/downloads/mysql/)
- **Discord Bot Token** - [Create one here](https://discord.com/developers/applications)

### Installation

1. **Clone the repository**
   ```bash
   git clone https://github.com/your-username/DiscordGo-GamblingBot.git
   cd DiscordGo-GamblingBot
   ```

2. **Install dependencies**
   ```bash
   go mod tidy
   ```

3. **Set up your MySQL database**
   ```sql
   CREATE DATABASE discord_gambling_bot;
   ```

4. **Configure environment variables**
   
   Create a `.env` file or set environment variables:
   ```bash
   export gamblingBotToken="your_discord_bot_token_here"
   export dbPath="username:password@tcp(localhost:3306)/discord_gambling_bot"
   ```

5. **Initialize database tables**
   
   On first run, uncomment this line in `main.go`:
   ```go
   if err := setupTables(db); err != nil {
       log.Fatal("Failed to setup tables:", err)
   }
   ```
   
   After the first successful run, you can comment it out again.

6. **Run the bot**
   ```bash
   go run .
   ```

---

## ⚙️ Configuration

### Required Environment Variables

| Variable | Description | Example |
|----------|-------------|---------|
| `gamblingBotToken` | Your Discord bot token | `MTIzNDU2Nzg5...` |
| `dbPath` | MySQL connection string | `root:password@tcp(127.0.0.1:3306)/discord_bot` |

### Code Configuration

#### Guild/Server IDs

Update these values in the source code to match your Discord server setup:

**In `slash.go`:**
```go
GuildIDs: []string{
    "YOUR_SERVER_ID_HERE",
},
```

**In `slot.go`:**
```go
guildIDs := []string{
    "YOUR_SERVER_ID_HERE", // Server where custom emojis are hosted
}
```

> **Note**: The bot must be a member of any servers listed in `slot.go` for custom slot emojis/GIFs to work properly.

---

## 🗄️ Database Schema

The bot automatically creates the following tables on first run:

<details>
<summary><strong>Click to expand database schema</strong></summary>

### Users Table
```sql
CREATE TABLE IF NOT EXISTS users (
    userid BIGINT UNSIGNED PRIMARY KEY,
    username VARCHAR(32) NOT NULL,
    balance DECIMAL(10,2) NOT NULL DEFAULT 0.00,
    wins DECIMAL(10,2) NOT NULL DEFAULT 0.00,
    losses DECIMAL(10,2) NOT NULL DEFAULT 0.00,
    admin TINYINT NOT NULL DEFAULT 0,
    banned TINYINT NOT NULL DEFAULT 0
);
```

### Active Games Table
```sql
CREATE TABLE IF NOT EXISTS active_games (
    id INT AUTO_INCREMENT PRIMARY KEY,
    userid BIGINT UNSIGNED,
    type VARCHAR(32) NOT NULL,
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
    FOREIGN KEY (userid) REFERENCES users(userid) ON DELETE CASCADE
);
```

### Game History Table
```sql
CREATE TABLE IF NOT EXISTS games (
    id INT AUTO_INCREMENT PRIMARY KEY,
    userid BIGINT UNSIGNED NOT NULL,
    game_type VARCHAR(32) NOT NULL,
    amount DECIMAL(10,2) NOT NULL,
    outcome DECIMAL(10,2) NOT NULL,
    played_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (userid) REFERENCES users(userid) ON DELETE CASCADE
);
```

### Transactions Table
```sql
CREATE TABLE IF NOT EXISTS transactions (
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
);
```

### Daily Rewards Table
```sql
CREATE TABLE IF NOT EXISTS daily_rewards (
    id INT AUTO_INCREMENT PRIMARY KEY,
    userid BIGINT UNSIGNED NOT NULL,
    claim_date DATE NOT NULL,
    streak INT NOT NULL,
    reward_amount DECIMAL(10,2) NOT NULL,
    claimed_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (userid) REFERENCES users(userid) ON DELETE CASCADE,
    UNIQUE(userid, claim_date)
);
```

</details>

---

## 🌐 Web Dashboard

Access the web dashboard at `http://localhost:8080` (default port) to view:

- Total users and active players
- Game statistics and popular games
- Economic health metrics
- Recent transactions
- System performance

To change the dashboard port, modify the configuration in `main.go`.

---

## 🤝 Contributing

Contributions are welcome! Here's how you can help:

1. Fork the repository
2. Create a feature branch (`git checkout -b feature/amazing-feature`)
3. Commit your changes (`git commit -m 'Add some amazing feature'`)
4. Push to the branch (`git push origin feature/amazing-feature`)
5. Open a Pull Request

Please ensure your code follows Go best practices and includes appropriate documentation.

---

## 📜 License

This project is licensed under the GPL-3.0 license License - see the LICENSE file for details.

---

## ⚠️ Disclaimer

This bot is for entertainment purposes only. No real money is involved. Please gamble responsibly, even with virtual currency.

---

## 📞 Support

- **Issues**: [GitHub Issues](https://github.com/tahawael/DiscordGo-GamblingBot/issues)

---

<p align="center">
  Made with ❤️ and Go
</p>

<p align="center">
  <a href="#top">Back to top ⬆️</a>
</p>
