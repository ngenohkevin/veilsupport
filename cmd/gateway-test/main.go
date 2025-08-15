package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"strings"
	"time"

	"github.com/ngenohkevin/veilsupport/internal/xmpp"
)

func main() {
	fmt.Println("🚀 VeilSupport XMPP Gateway Test")
	fmt.Println("=================================")
	fmt.Println()
	fmt.Println("This test demonstrates how website users appear as separate contacts in Conversations")
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
	
	// Multiple admins can be separated by comma
	adminJIDs := []string{adminJID}
	if strings.Contains(adminJID, ",") {
		adminJIDs = strings.Split(adminJID, ",")
		for i := range adminJIDs {
			adminJIDs[i] = strings.TrimSpace(adminJIDs[i])
		}
	}
	
	fmt.Printf("📋 Gateway Configuration:\n")
	fmt.Printf("  Bot JID: %s\n", botJID)
	fmt.Printf("  Server: %s\n", xmppServer)
	fmt.Printf("  Admin JIDs: %v\n", adminJIDs)
	fmt.Println()
	
	// Create gateway client
	gateway := xmpp.NewGatewayClient(botJID, botPassword, xmppServer, adminJIDs)
	
	// Connect
	fmt.Println("🔌 Connecting gateway to XMPP server...")
	ctx := context.Background()
	err := gateway.Connect(ctx)
	if err != nil {
		log.Fatalf("Failed to connect gateway: %v", err)
	}
	fmt.Println("✅ Gateway connected successfully!")
	fmt.Println()
	
	// Simulate multiple website users
	users := []struct {
		ID          int
		Email       string
		DisplayName string
		Messages    []string
	}{
		{
			ID:          101,
			Email:       "john.doe@example.com",
			DisplayName: "John Doe",
			Messages: []string{
				"Hello, I need help with my order #12345",
				"It hasn't arrived yet and it's been 5 days",
				"Can you check the status please?",
			},
		},
		{
			ID:          102,
			Email:       "jane.smith@example.com",
			DisplayName: "Jane Smith",
			Messages: []string{
				"Hi, I'd like to return an item",
				"The product is defective",
				"How do I get a refund?",
			},
		},
		{
			ID:          103,
			Email:       "bob.wilson@example.com",
			DisplayName: "Bob Wilson",
			Messages: []string{
				"Is this product still in stock?",
				"I need 10 units for my business",
				"Can you offer a bulk discount?",
			},
		},
	}
	
	fmt.Println("👥 Simulating messages from 3 different website users...")
	fmt.Println("Each user will appear as a SEPARATE contact in Conversations!")
	fmt.Println()
	
	for _, user := range users {
		fmt.Printf("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━\n")
		fmt.Printf("👤 User: %s (%s)\n", user.DisplayName, user.Email)
		fmt.Printf("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━\n")
		
		// Register user with gateway
		resourceID := gateway.RegisterUser(user.ID, user.Email, user.DisplayName)
		fmt.Printf("📝 Registered as: %s\n", resourceID)
		
		// Set user online
		gateway.SetUserOnline(user.ID, true)
		fmt.Printf("🟢 User is now online\n")
		
		// Send messages from this user
		for i, msg := range user.Messages {
			fmt.Printf("\n📨 Message %d: %s\n", i+1, msg)
			
			// Simulate file attachment for some messages
			var attachments []string
			if i == 0 && user.ID == 101 {
				attachments = []string{"https://example.com/order-screenshot.png"}
				fmt.Println("   📎 With attachment: order-screenshot.png")
			}
			
			err := gateway.SendUserMessage(user.ID, msg, attachments)
			if err != nil {
				fmt.Printf("   ❌ Failed: %v\n", err)
			} else {
				fmt.Printf("   ✅ Sent to admins\n")
			}
			
			// Small delay between messages
			time.Sleep(2 * time.Second)
		}
		
		// Set user offline after sending messages
		gateway.SetUserOnline(user.ID, false)
		fmt.Printf("\n🔴 User is now offline\n")
		fmt.Println()
		
		// Delay between users
		time.Sleep(3 * time.Second)
	}
	
	fmt.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
	fmt.Println()
	fmt.Println("✨ Test Complete!")
	fmt.Println()
	fmt.Println("📱 Check your Conversations app:")
	fmt.Println("   You should see messages from 3 DIFFERENT users:")
	fmt.Println("   • John Doe (john.doe@example.com)")
	fmt.Println("   • Jane Smith (jane.smith@example.com)")
	fmt.Println("   • Bob Wilson (bob.wilson@example.com)")
	fmt.Println()
	fmt.Println("💬 Each user appears as a separate conversation thread!")
	fmt.Println("   You can reply to each user individually")
	fmt.Println()
	fmt.Println("📝 To reply to a specific user from Conversations:")
	fmt.Println("   Start your message with @user_ID")
	fmt.Println("   Example: @user_101 Your order is on the way!")
	fmt.Println()
	
	// Keep connection alive for a bit
	fmt.Println("Keeping gateway connection alive for 30 seconds...")
	fmt.Println("(You can test replying from Conversations during this time)")
	time.Sleep(30 * time.Second)
	
	// Close connection
	fmt.Println("\nClosing gateway connection...")
	err = gateway.Close()
	if err != nil {
		fmt.Printf("Warning: Error closing connection: %v\n", err)
	}
	
	fmt.Println("👋 Goodbye!")
}
