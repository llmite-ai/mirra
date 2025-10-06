#!/bin/bash
set -euo pipefail
bundle install
bundle exec ruby main.rb
