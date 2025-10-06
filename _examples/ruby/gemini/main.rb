#!/usr/bin/env ruby
require 'gemini'

def main
  api_key = ENV['GEMINI_API_KEY']
  unless api_key
    warn 'Error: GEMINI_API_KEY environment variable not set'
    exit 1
  end

  begin
    client = Gemini::Client.new(
      api_key,
      uri_base: 'http://localhost:4567' # This is where we're configuring mirra
    )

    response = client.generate_content(
      'Say hello and a joke',
      model: 'gemini-2.0-flash-exp'
    )

    if response.valid?
      puts response.text
    else
      puts 'No valid response from Gemini.'
    end

  rescue StandardError => e
    warn "Error: #{e.message}"
    exit 1
  end
end

main if __FILE__ == $PROGRAM_NAME
