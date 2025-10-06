#!/usr/bin/env ruby
require 'openai'

def main
  api_key = ENV['OPENAI_API_KEY']
  unless api_key
    warn 'Error: OPENAI_API_KEY environment variable not set'
    exit 1
  end

  begin
    client = OpenAI::Client.new(
      access_token: api_key,
      uri_base: 'http://localhost:4567/v1' # This is where we're configuring mirra
    )

    response = client.chat(
      parameters: {
        model: 'gpt-4o',
        messages: [
          { role: 'user', content: 'Say hello and a joke' }
        ]
      }
    )

    if response['choices'].nil? || response['choices'].empty?
      puts 'No response from OpenAI.'
      return
    end

    response['choices'].each do |choice|
      puts choice.dig('message', 'content')
    end

  rescue StandardError => e
    warn "Error: #{e.message}"
    exit 1
  end
end

main if __FILE__ == $PROGRAM_NAME
