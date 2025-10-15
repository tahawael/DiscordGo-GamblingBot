# DiscordGo-GamblingBot

Gambling bot for Discord, written in Go. It features an economy system, multiple games, and uses a MySQL database to store user data and game history.

## Features

  -  **Economy System**: Users have balances, can earn money, and transfer funds.
  -  **Multiple Games**: Currently Includes Mines and Slots.
  -  **Persistent Stats**: Tracks user wins, losses, and game history.
  -  **Daily Rewards**: Users can claim a daily reward with a streak system.
  -  **Admin Controls**: Includes basic admin functionality like banning users.
  -  **Web Dashboard**: A simple dashboard to view bot statistics.

-----

## Tech Stack

  - **Language**: [Go](https://golang.org/)
  - **Database**: [MySQL](https://www.mysql.com/)
  - **Discord Library**: [discordgo](https://github.com/bwmarrin/discordgo)

-----

## Installation & Setup

Follow these steps to get the bot up and running.

### 1\. Prerequisites

  - Go (version 1.18 or higher)
  - MySQL Server

### 2\. Clone Repository

```bash
git clone https://github.com/your-username/your-repo-name.git
cd your-repo-name
```

### 3\. Database Setup

1.  Create a new database in your MySQL server.

2.  The bot is designed to create the necessary tables automatically. In `main.go`, uncomment the call to `setupTables(db)` to initialize your database on the first run.

    ```go
    // main.go

    // ...
    log.Println("Database connected successfully")

    // only run when setting up the db
    if err := setupTables(db); err != nil {
     log.Fatal("Failed to setup tables:", err)
    }
    // ...
    ```

### 4\. Configuration

You need to set two environment variables for the bot to connect to Discord and your database.

  - `gamblingBotToken`: Your Discord bot token.
  - `dbPath`: Your MySQL connection string.
      - *Format*: `user:password@tcp(host:port)/dbname`
      - *Example*: `root:password@tcp(127.0.0.1:3306)/discord_bot`

You also need to update the Guild (Server) IDs in the source code:

  - In `slash.go`, update `GuildIDs` with your Discord server ID(s) for registering slash commands. (Optional if using global commands)
  - In `slot.go`, update `guildIDs` with the server ID(s) where custom slot GIFs/emojis are hosted. The bot must be a member of these servers.

### 5\. Run the Bot

1.  Install the required Go modules:
    ```bash
    go mod tidy
    ```
2.  Run the application:
    ```bash
    go run .
    ```

-----

## Database Schema

<details> <summary>Click to view MySQL Schemas</summary>
  
**Users**

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

**Active Games**

```sql
CREATE TABLE IF NOT EXISTS active_games (
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
    FOREIGN KEY (userid) REFERENCES users(userid) ON DELETE CASCADE
);
```

**Game History**

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

**Transactions**

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

**Daily Rewards**

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
