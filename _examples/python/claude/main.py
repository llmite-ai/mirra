import os
import sys
from anthropic import Anthropic


def main():
    api_key = os.getenv("ANTHROPIC_API_KEY")
    if not api_key:
        print("Error: ANTHROPIC_API_KEY environment variable not set", file=sys.stderr)
        sys.exit(1)

    try:
        client = Anthropic(
            api_key=api_key,
            base_url="http://localhost:4567"  # This is where we're configuring mirra
        )

        response = client.messages.create(
            model="claude-3-5-sonnet-20241022",
            max_tokens=1024,
            messages=[
                {"role": "user", "content": "Say hello and a joke"}
            ]
        )

        if not response.content:
            print("No response from Claude.")
            return

        for block in response.content:
            if block.type == "text":
                print(block.text)

    except Exception as e:
        print(f"Error: {e}", file=sys.stderr)
        sys.exit(1)


if __name__ == "__main__":
    main()
