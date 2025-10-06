import os
import sys
from openai import OpenAI


def main():
    api_key = os.getenv("OPENAI_API_KEY")
    if not api_key:
        print("Error: OPENAI_API_KEY environment variable not set", file=sys.stderr)
        sys.exit(1)

    try:
        client = OpenAI(
            api_key=api_key,
            base_url="http://localhost:4567/v1"  # This is where we're configuring mirra
        )

        response = client.chat.completions.create(
            model="gpt-4o",
            messages=[
                {"role": "user", "content": "Say hello and a joke"}
            ]
        )

        if not response.choices:
            print("No response from OpenAI.")
            return

        for choice in response.choices:
            print(choice.message.content)

    except Exception as e:
        print(f"Error: {e}", file=sys.stderr)
        sys.exit(1)


if __name__ == "__main__":
    main()
