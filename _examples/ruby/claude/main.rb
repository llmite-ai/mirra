#!/usr/bin/env ruby
require 'anthropic'

def main
  api_key = ENV['ANTHROPIC_API_KEY']
  unless api_key
    warn 'Error: ANTHROPIC_API_KEY environment variable not set'
    exit 1
  end

  begin
    client = Anthropic::Client.new(
      access_token: api_key,
      api_url: 'http://localhost:4567' # This is where we're configuring mirra
    )

    response = client.messages(parameters: {
      model: 'claude-3-5-sonnet-20241022',
      max_tokens: 1024,
      messages: [
        { role: 'user', content: 'Say hello and a joke' }
      ]
    })

    content = response['content'] || response[:content]

    if content.nil? || content.empty?
      puts 'No response from Claude.'
      return
    end

    content.each do |block|
      if block.is_a?(Hash) && (block['text'] || block[:text])
        puts block['text'] || block[:text]
      else
        puts block.to_s
      end
    end

  rescue StandardError => e
    warn "Error: #{e.message}"
    exit 1
  end
end

main if __FILE__ == $PROGRAM_NAME
