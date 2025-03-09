#!/usr/bin/env bash

# Check if curl is installed
if ! command -v curl &> /dev/null; then
    echo "curl could not be found, please install it."
    exit 1
fi

# Check if jq is installed
if ! command -v jq &> /dev/null; then
    echo "jq could not be found, please install it."
    exit 1
fi

# Check if LOGIN and PASSWORD are provided
if [ -n "$1" ] && [ -n "$2" ]; then
    LOGIN="$1"
    PASSWORD="$2"

    # Fetch the token for authenticated user
    TOKEN=$(curl -s --user "$LOGIN:$PASSWORD" "https://auth.docker.io/token?service=registry.docker.io&scope=repository:ratelimitpreview/test:pull" | jq -r .token)

    # Check if the token was successfully retrieved
    if [ -z "$TOKEN" ]; then
        echo "Failed to retrieve token. Please check your LOGIN and PASSWORD."
        exit 1
    fi
else
    TOKEN=$(curl -s "https://auth.docker.io/token?service=registry.docker.io&scope=repository:ratelimitpreview/test:pull" | jq -r .token)
    # Check if the token was successfully retrieved
    if [ -z "$TOKEN" ]; then
        echo "Failed to retrieve token. Please check your LOGIN and PASSWORD."
        exit 1
    fi
fi

# Fetch the rate limit information for authenticated user
RESPONSE=$(curl -s --head -H "Authorization: Bearer $TOKEN" https://registry-1.docker.io/v2/ratelimitpreview/test/manifests/latest)

# Check if the request was successful
if [ -z "$RESPONSE" ]; then
    echo "Failed to fetch rate limit information."
    exit 1
fi

# Check for rate limit headers
RATELIMIT_LIMIT=$(echo "$RESPONSE" | grep -i "ratelimit-limit" || true)
RATELIMIT_REMAINING=$(echo "$RESPONSE" | grep -i "ratelimit-remaining" || true)

# Display the rate limit information
if [ -n "$RATELIMIT_LIMIT" ] || [ -n "$RATELIMIT_REMAINING" ]; then
    echo "Rate Limit Information:"
    echo "$RATELIMIT_LIMIT"
    echo "$RATELIMIT_REMAINING"
else
    echo "No rate limit information found in the response. This might be a business account or the headers are not exposed."
    echo "Full response headers for debugging:"
    echo "$RESPONSE"
fi
