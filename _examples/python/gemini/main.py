import os
import sys
import google.generativeai as genai


def main():
    api_key = os.getenv("GEMINI_API_KEY")
    if not api_key:
        print("Error: GEMINI_API_KEY environment variable not set", file=sys.stderr)
        sys.exit(1)

    try:
        genai.configure(
            api_key=api_key,
            transport="rest",
            client_options={
                "api_endpoint": "http://localhost:4567"  # This is where we're configuring mirra
            }
        )

        model = genai.GenerativeModel("gemini-2.0-flash-exp")
        response = model.generate_content("Say hello and a joke")

        if not response.text:
            print("No response from Gemini.")
            return

        print(response.text)

    except Exception as e:
        print(f"Error: {e}", file=sys.stderr)
        sys.exit(1)


if __name__ == "__main__":
    main()
