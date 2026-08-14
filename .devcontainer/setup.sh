#!/bin/bash

set -e

echo "Setting up Go development environment with bakito/toolbox..."

# Update and install essential tools
sudo apt-get install -y curl wget make

# Install Go tools
go install golang.org/x/tools/gopls@latest
go install github.com/go-delve/delve/cmd/dlv@latest
go install honnef.co/go/tools/cmd/staticcheck@latest
