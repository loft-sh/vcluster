#!/bin/bash

set -x

LIST_URL="https://raw.githubusercontent.com/disposable-email-domains/disposable-email-domains/refs/heads/main/disposable_email_blocklist.conf"
URL_LIST="disposable_domains.conf"
JSON_FILE="disposable_domains.json"
ZIP_FILE="disposable_domains.zip"

if ! curl -o "$URL_LIST" "$LIST_URL"; then
    echo "Failed to download the disposable domains file."
    exit 1
fi

jq -R -s 'split("\n") | map(select(length > 0))' "$URL_LIST" > "$JSON_FILE"

zip "$ZIP_FILE" "$JSON_FILE"

rm "$URL_LIST" "$JSON_FILE"

mv "$ZIP_FILE" ../../pkg/cli/email

echo "Successfully created $ZIP_FILE"
