package main

import (
	"bufio"
	"context"
	"fmt"
	"log"
	"os"
	"strings"
	"time"

	"github.com/ngenohkevin/veilsupport/internal/xmpp"
)

func main() {
	fmt.Println("🤖 VeilSupport - Realistic Single Conversation Bot")
	fmt.Println("══════════════════════════════════════════════════")
	fmt.Println()
	fmt.Println("This demonstrates the REALISTIC implementation:")
	fmt.Println("• All users appear in ONE conversation (XMPP limitation)")
	fmt.Println("• But each message is clearly formatted and labeled")
	fmt.Println("• Easy reply system using @USER_ID format")
	fmt.Println()
	
	// Load configuration
	botJID := os.Getenv("XMPP_CONNECTION_JID")
	if botJID == "" {
		log.Fatal("XMPP_CONNECTION_JID not set")
	}
	
	botPassword := os.Getenv("XMPP_CONNECTION_PASSWORD")
	if botPassword == "" {
		log.Fatal("XMPP_CONNECTION_PASSWORD not set")
	}
	
	xmppServer := os.Getenv("XMPP_SERVER")
	if xmppServer == "" {
		xmppServer = "xmpp.jp:5222"
	}
	
	adminJID := os.Getenv("XMPP_ADMIN_JID")
	if adminJID == "" {
		log.Fatal("XMPP_ADMIN_JID not set")
	}
	
	fmt.Printf("📋 Configuration:\n")
	fmt.Printf("  Bot JID: %s\n", botJID)
	fmt.Printf("  Admin JID: %s\n", adminJID)
	fmt.Printf("  Server: %s\n", xmppServer)
	fmt.Println()
	
	// Create better bot
	bot := xmpp.NewBetterBotClient(botJID, botPassword, xmppServer, adminJID)
	
	// Connect
	fmt.Println("🔌 Connecting to XMPP server...")
	ctx := context.Background()
	err := bot.Connect(ctx)
	if err != nil {
		log.Fatalf("Failed to connect: %v", err)
	}
	fmt.Println("✅ Connected successfully!")
	fmt.Println()
	
	// Simulate different website users
	users := []struct {
		ID       int
		Email    string
		Name     string
		Messages []string
	}{
		{
			ID:    101,
			Email: "john.doe@example.com",
			Name:  "John Doe",
			Messages: []string{
				"Hello, I need help with my order #12345",
				"It hasn't arrived yet and it's been 5 days",
				"Can you check the status please?",
			},
		},
		{
			ID:    102,
			Email: "jane.smith@example.com",
			Name:  "Jane Smith",
			Messages: []string{
				"Hi, I'd like to return an item",
				"The product is defective",
				"How do I get a refund?",
			},
		},
		{
			ID:    103,
			Email: "bob.wilson@example.com",
			Name:  "Bob Wilson",
			Messages: []string{
				"Is this product still in stock?",
				"I need 10 units for my business",
				"Can you offer a bulk discount?",
			},
		},
	}
	
	fmt.Println("📨 Simulating messages from 3 different website users...")
	fmt.Println("All will appear in ONE conversation, but clearly formatted")
	fmt.Println()
	
	// Send initial messages
	for _, user := range users {
		fmt.Printf("Sending messages from %s (User #%d)...\n", user.Name, user.ID)
		
		for _, msg := range user.Messages {
			err := bot.SendUserMessage(user.ID, user.Email, user.Name, msg)
			if err != nil {
				fmt.Printf("  ❌ Error: %v\n", err)
			} else {
				fmt.Printf("  ✅ Sent: %s\n", msg)
			}
			time.Sleep(2 * time.Second)
		}
		fmt.Println()
		time.Sleep(3 * time.Second)
	}
	
	// Send list command
	fmt.Println("📋 Sending /list command to show active users...")
	bot.HandleCommand("/list")
	time.Sleep(2 * time.Second)
	
	// Interactive mode
	fmt.Println()
	fmt.Println("══════════════════════════════════════════════════")
	fmt.Println("📱 CHECK YOUR CONVERSATIONS APP NOW!")
	fmt.Println("You'll see all messages in ONE conversation with veil_support")
	fmt.Println("But each message is clearly formatted with user info")
	fmt.Println()
	fmt.Println("🎮 INTERACTIVE MODE - Test admin commands:")
	fmt.Println("  /list - Show active users")
	fmt.Println("  /info USER_ID - Get user details")
	fmt.Println("  /help - Show available commands")
	fmt.Println("  @USER_ID message - Reply to a user")
	fmt.Println("  quit - Exit program")
	fmt.Println("══════════════════════════════════════════════════")
	fmt.Println()
	
	scanner := bufio.NewScanner(os.Stdin)
	for {
		fmt.Print("Admin command> ")
		if !scanner.Scan() {
			break
		}
		
		input := strings.TrimSpace(scanner.Text())
		
		if input == "quit" || input == "exit" {
			break
		}
		
		if input == "" {
			continue
		}
		
		// Handle command or reply
		err := bot.HandleCommand(input)
		if err != nil {
			fmt.Printf("Error: %v\n", err)
		}
		
		// If it's a reply, also simulate sending it to the user
		if strings.HasPrefix(input, "@") {
			userID, reply, err := bot.ParseAdminReply(input)
			if err == nil {
				fmt.Printf("💬 Reply would be sent to User #%d via WebSocket: %s\n", 
					userID, reply)
			}
		}
	}
	
	fmt.Println()
	fmt.Println("Closing connection...")
	err = bot.Close()
	if err != nil {
		fmt.Printf("Warning: Error closing connection: %v\n", err)
	}
	
	fmt.Println("👋 Goodbye!")
}
