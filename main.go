package main

import (
	"fmt"
	"log"
	"os"
	"strings"

	"github.com/bwmarrin/discordgo"
	"github.com/joho/godotenv"
	
	"discordxel/commands"
)

func main() {
	// Load .env file
	if err := godotenv.Load(); err != nil {
		log.Fatal("Error loading .env file")
	}

	// Get Discord token from environment variable
	token := os.Getenv("DISCORD_TOKEN")
	if token == "" {
		log.Fatal("Discord token not found in .env file")
	}

	// Create new Discord session
	dg, err := discordgo.New("Bot " + token)
	if err != nil {
		log.Fatal("Error creating Discord session:", err)
	}

	// Register messageCreate as a callback for the messageCreate events
	dg.AddHandler(messageCreate)

	// Open a websocket connection to Discord
	err = dg.Open()
	if err != nil {
		log.Fatal("Error opening connection:", err)
	}

	fmt.Println("Bot is now running. Press CTRL-C to exit.")
	sc := make(chan os.Signal, 1)
	<-sc

	// Close Discord session
	dg.Close()
}

func messageCreate(s *discordgo.Session, m *discordgo.MessageCreate) {
	// Ignore messages from the bot itself
	if m.Author.ID == s.State.User.ID {
		return
	}

	// Check if message starts with "?"
	if !strings.HasPrefix(m.Content, "?") {
		return
	}

	// Split the message into command and arguments
	args := strings.Fields(m.Content[1:])
	if len(args) == 0 {
		return
	}

	command := strings.ToLower(args[0])

	// Handle commands
	switch command {
	case "help", "h", "bantuan", "?":
		commands.HandleFinanceCommand(s, m, "help", args)
	case "uangmasuk", "um", "in", "masuk":
		commands.HandleFinanceCommand(s, m, "uangmasuk", args)
	case "uangkeluar", "uk", "out", "keluar":
		commands.HandleFinanceCommand(s, m, "uangkeluar", args)
	case "totaluang", "tu", "total", "saldo":
		commands.HandleFinanceCommand(s, m, "totaluang", args)
	case "cleartransaksi", "ct", "clear", "hapus":
		commands.HandleFinanceCommand(s, m, "cleartransaksi", args)
	case "confirmclear", "cc", "confirm":
		commands.HandleFinanceCommand(s, m, "confirmclear", args)
	}
}
