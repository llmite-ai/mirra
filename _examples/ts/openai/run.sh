#!/bin/bash

set -e

# Check if OPENAI_API_KEY is set
if [ -z "$OPENAI_API_KEY" ]; then
  echo "Error: OPENAI_API_KEY environment variable is not set"
  exit 1
fi

# Install dependencies
echo "Installing dependencies..."
npm install

# Run the example
echo "Running example..."
npm start
