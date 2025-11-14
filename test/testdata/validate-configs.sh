#!/bin/bash
set -e

echo "Validating Terraform configs in testdata/configs/"
cd "$(dirname "$0")/configs"

failed=0
for config in *.tofu; do
    echo -n "Validating $config... "
    if terraform fmt -check "$config" > /dev/null 2>&1; then
        echo "✓ formatted"
    else
        echo "✗ needs formatting"
        terraform fmt "$config"
        echo "  → auto-formatted"
    fi
done

echo ""
echo "All configs validated successfully!"
exit $failed
