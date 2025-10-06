# Gemini Java Example with Mirra

This example demonstrates how to use the Google Gemini API with the Mirra proxy.

## Prerequisites

- Java 11 or higher installed
- `GEMINI_API_KEY` environment variable set
- Mirra proxy running on `http://localhost:4567`

## Setup

If you're using asdf version manager, install Java:

```bash
asdf plugin add java
asdf install java openjdk-11.0.2
```

## Running

```bash
./run.sh
```

Or via make from the repository root:

```bash
make run_example java gemini
```

## How it works

The example:
1. Checks for the `GEMINI_API_KEY` environment variable
2. Makes an HTTP request to the Gemini API through the Mirra proxy (`http://localhost:4567/v1`)
3. Sends a simple prompt: "Say hello and a joke"
4. Prints the response

The Maven wrapper (`./mvnw`) is included, so you don't need Maven pre-installed.
