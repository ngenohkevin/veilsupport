package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/ngenohkevin/veilsupport/internal/xmpp"
)

func main() {
	// Load XMPP configuration from environment
	xmppServer := os.Getenv("XMPP_SERVER")
	if xmppServer == "" {
		xmppServer = "xmpp.jp:5222"
	}
	
	xmppJID := os.Getenv("XMPP_CONNECTION_JID")
	if xmppJID == "" {
		log.Fatal("XMPP_CONNECTION_JID not set")
	}
	
	xmppPassword := os.Getenv("XMPP_CONNECTION_PASSWORD")
	if xmppPassword == "" {
		log.Fatal("XMPP_CONNECTION_PASSWORD not set")
	}
	
	adminJID := os.Getenv("XMPP_ADMIN_JID")
	if adminJID == "" {
		log.Fatal("XMPP_ADMIN_JID not set")
	}
	
	fmt.Printf("Testing XMPP connection:\n")
	fmt.Printf("  Server: %s\n", xmppServer)
	fmt.Printf("  From JID: %s\n", xmppJID)
	fmt.Printf("  To JID: %s\n", adminJID)
	fmt.Println()
	
	// Create XMPP client
	client := xmpp.NewXMPPClient(xmppJID, xmppPassword, xmppServer)
	
	// Connect
	fmt.Println("Connecting to XMPP server...")
	ctx := context.Background()
	err := client.ConnectWithContext(ctx)
	if err != nil {
		log.Fatalf("Failed to connect: %v", err)
	}
	fmt.Println("✓ Connected successfully!")
	
	// Wait a moment for the connection to stabilize
	time.Sleep(2 * time.Second)
	
	// Send test messages
	messages := []string{
		"Test message 1: Hello from VeilSupport!",
		"Test message 2: User needs help with order #123",
		"Test message 3: Connection test successful ✓",
	}
	
	for i, msg := range messages {
		fmt.Printf("\nSending message %d...\n", i+1)
		
		// Try the regular send method first
		err = client.SendMessage(adminJID, msg)
		if err != nil {
			fmt.Printf("Regular send failed: %v\n", err)
			
			// Try the simple send method
			fmt.Println("Trying simple send method...")
			err = client.SendMessageSimple(adminJID, msg)
			if err != nil {
				fmt.Printf("Simple send also failed: %v\n", err)
			} else {
				fmt.Printf("✓ Message sent via simple method: %s\n", msg)
			}
		} else {
			fmt.Printf("✓ Message sent: %s\n", msg)
		}
		
		// Wait between messages
		time.Sleep(1 * time.Second)
	}
	
	fmt.Println("\n✓ Test complete! Check your Conversations app for the messages.")
	fmt.Println("The messages should appear from:", xmppJID)
	fmt.Println("In the conversation with:", adminJID)
	
	// Keep connection alive for a bit
	fmt.Println("\nKeeping connection alive for 10 seconds...")
	time.Sleep(10 * time.Second)
	
	// Close connection
	fmt.Println("Closing connection...")
	err = client.Close()
	if err != nil {
		fmt.Printf("Warning: Error closing connection: %v\n", err)
	}
	
	fmt.Println("Done!")
}
