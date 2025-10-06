#!/usr/bin/env ruby
require 'gemini-ai'

# Monkey patch to support custom base address for proxy
module Gemini
  module Controllers
    class Client
      def initialize(config)
        @service = config[:credentials][:service]

        unless %w[vertex-ai-api generative-language-api].include?(@service)
          raise Errors::UnsupportedServiceError, "Unsupported service: '#{@service}'."
        end

        avoid_conflicting_credentials!(config[:credentials])

        if config[:credentials][:api_key]
          @authentication = :api_key
          @api_key = config[:credentials][:api_key]
        elsif config[:credentials][:file_path] || config[:credentials][:file_contents]
          @authentication = :service_account
          json_key_io = if config[:credentials][:file_path]
                          File.open(config[:credentials][:file_path])
                        else
                          StringIO.new(config[:credentials][:file_contents])
                        end

          @authorizer = ::Google::Auth::ServiceAccountCredentials.make_creds(
            json_key_io:,
            scope: 'https://www.googleapis.com/auth/cloud-platform'
          )
        else
          @authentication = :default_credentials
          @authorizer = ::Google::Auth.get_application_default
        end

        if @authentication == :service_account || @authentication == :default_credentials
          @project_id = config[:credentials][:project_id] || @authorizer.project_id || @authorizer.quota_project_id

          raise Errors::MissingProjectIdError, 'Could not determine project_id, which is required.' if @project_id.nil?
        end

        @service_version = config.dig(:credentials, :version) || DEFAULT_SERVICE_VERSION

        # Support custom base address for proxy
        if config.dig(:credentials, :base_address)
          @base_address = config[:credentials][:base_address]
        else
          @base_address = case @service
                          when 'vertex-ai-api'
                            "https://#{config[:credentials][:region]}-aiplatform.googleapis.com/#{@service_version}/projects/#{@project_id}/locations/#{config[:credentials][:region]}"
                          when 'generative-language-api'
                            "https://generativelanguage.googleapis.com/#{@service_version}"
                          end
        end

        @model_address = case @service
                         when 'vertex-ai-api'
                           "publishers/google/models/#{config[:options][:model]}"
                         when 'generative-language-api'
                           "models/#{config[:options][:model]}"
                         end

        @server_sent_events = config.dig(:options, :server_sent_events)

        @request_options = config.dig(:options, :connection, :request)

        @faraday_adapter = config.dig(:options, :connection, :adapter) || DEFAULT_FARADAY_ADAPTER

        @request_options = if @request_options.is_a?(Hash)
                             @request_options.select do |key, _|
                               ALLOWED_REQUEST_OPTIONS.include?(key)
                             end
                           else
                             {}
                           end
      end
    end
  end
end

def main
  api_key = ENV['GEMINI_API_KEY']
  unless api_key
    warn 'Error: GEMINI_API_KEY environment variable not set'
    exit 1
  end

  begin
    client = Gemini.new(
      credentials: {
        service: 'generative-language-api',
        api_key: api_key,
        version: 'v1beta', # Required for streaming API
        base_address: 'http://localhost:4567/v1beta' # Mirra proxy URL
      },
      options: {
        model: 'gemini-2.0-flash-exp',
        server_sent_events: true
      }
    )

    puts "Making request through Mirra proxy..."

    client.stream_generate_content(
      { contents: { role: 'user', parts: { text: 'Say hello and tell me a joke' } } }
    ) do |event, _parsed, _raw|
      if event.dig('candidates', 0, 'content', 'parts', 0, 'text')
        print event['candidates'][0]['content']['parts'][0]['text']
      end
    end

    puts "\n"

  rescue StandardError => e
    warn "Error: #{e.message}"
    warn e.backtrace.join("\n")
    exit 1
  end
end

main if __FILE__ == $PROGRAM_NAME
