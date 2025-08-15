#!/bin/bash

# VeilSupport XMPP Quick Test Script
# This script helps you quickly test the XMPP integration

echo "🚀 VeilSupport XMPP Integration Test"
echo "===================================="
echo ""

# Check if .env exists
if [ ! -f .env ]; then
    echo "❌ Error: .env file not found!"
    echo "Please copy .env.example to .env and configure your XMPP settings"
    exit 1
fi

# Load environment variables
source .env

echo "📋 XMPP Configuration:"
echo "  Server: $XMPP_SERVER"
echo "  From: $XMPP_CONNECTION_JID"
echo "  To: $XMPP_ADMIN_JID"
echo ""

# Ask user what to do
echo "What would you like to do?"
echo "1) Test XMPP connection only"
echo "2) Run the full server"
echo "3) Send a test message via API"
echo ""
read -p "Enter choice (1-3): " choice

case $choice in
    1)
        echo ""
        echo "🔄 Testing XMPP connection..."
        go run cmd/xmpp-test/main.go
        ;;
    2)
        echo ""
        echo "🚀 Starting VeilSupport server..."
        echo "Server will be available at http://localhost:$PORT"
        echo "Press Ctrl+C to stop"
        echo ""
        go run cmd/server/main.go
        ;;
    3)
        echo ""
        echo "📨 Sending test message via API..."
        echo ""
        
        # Check if server is running
        if ! curl -s http://localhost:$PORT > /dev/null 2>&1; then
            echo "❌ Server is not running. Please start it first (option 2)"
            exit 1
        fi
        
        # Register test user
        echo "1. Registering test user..."
        REGISTER_RESPONSE=$(curl -s -X POST http://localhost:$PORT/api/register \
            -H "Content-Type: application/json" \
            -d '{"email":"test@example.com","password":"password123"}')
        
        # Extract token (simple grep since jq might not be installed)
        TOKEN=$(echo $REGISTER_RESPONSE | grep -o '"token":"[^"]*' | grep -o '[^"]*$')
        
        if [ -z "$TOKEN" ]; then
            # Try login if registration failed (user might exist)
            echo "   User might exist, trying login..."
            LOGIN_RESPONSE=$(curl -s -X POST http://localhost:$PORT/api/login \
                -H "Content-Type: application/json" \
                -d '{"email":"test@example.com","password":"password123"}')
            TOKEN=$(echo $LOGIN_RESPONSE | grep -o '"token":"[^"]*' | grep -o '[^"]*$')
        fi
        
        if [ -z "$TOKEN" ]; then
            echo "❌ Failed to get auth token"
            exit 1
        fi
        
        echo "   ✓ Got auth token"
        echo ""
        
        # Send message
        echo "2. Sending test message..."
        curl -X POST http://localhost:$PORT/api/send \
            -H "Authorization: Bearer $TOKEN" \
            -H "Content-Type: application/json" \
            -d '{"message":"Test message from VeilSupport API!"}'
        
        echo ""
        echo ""
        echo "✅ Message sent! Check your Conversations app"
        echo "   The message should appear from: $XMPP_CONNECTION_JID"
        echo "   In conversation with: $XMPP_ADMIN_JID"
        ;;
    *)
        echo "Invalid choice"
        exit 1
        ;;
esac

echo ""
echo "✨ Done!"
