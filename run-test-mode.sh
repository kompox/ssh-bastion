#!/bin/bash
# Quick start script for local browser testing

set -e

echo "🚀 Starting SSH Bastion in TEST MODE"
echo ""

# Set up environment
export SSHBASTION_DATA_DIR="_tmp/data"
export SSHBASTION_AUTH_MODE="easy_auth"
export SSHBASTION_AUTH_OVERRIDE_USER_ID="test-user-123"
export SSHBASTION_AUTH_OVERRIDE_EMAIL="developer@localhost"

# Create data directory
mkdir -p _tmp/data

echo "Configuration:"
echo "  Data directory: $SSHBASTION_DATA_DIR"
echo "  Auth mode: $SSHBASTION_AUTH_MODE"
echo "  Test user: $SSHBASTION_AUTH_OVERRIDE_USER_ID"
echo "  Test email: $SSHBASTION_AUTH_OVERRIDE_EMAIL"
echo ""
echo "⚠️  WARNING: This is TEST MODE - do not use in production!"
echo ""
echo "You can now open your browser to: http://localhost:8080/"
echo ""
echo "Press Ctrl+C to stop the server"
echo ""

# Run the server
./ssh-bastion web
