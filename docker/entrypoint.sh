#!/bin/bash
set -e

# Install any custom CA certificates that have been mounted
if [ -d "/usr/local/share/ca-certificates" ] && [ "$(ls -A /usr/local/share/ca-certificates 2>/dev/null)" ]; then
    # Run update-ca-certificates to install mounted CA certs into the system trust store
    # This makes them available to all applications (curl, wget, git, etc.) and subshells
    if ! update-ca-certificates --fresh 2>&1; then
        echo "Warning: update-ca-certificates failed, some CA certificates may not be available" >&2
    fi
fi

# Execute the main command (claude)
exec /usr/local/bin/claude "$@"
