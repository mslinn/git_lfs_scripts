#!/bin/bash
# Test script for remote mode functionality

set -e

echo "=== Remote Mode Test Script ==="
echo

# Clean up
rm -f test-remote.db test-remote.json

echo "1. Creating test database and run..."
./bin/lfst-run --db test-remote.db create \
    --scenario 6 \
    --server lfs-test-server \
    --protocol http \
    --notes "Remote mode test"

# Get the run ID (it will be 1 since this is a fresh database)
RUN_ID=1
echo "   Created test run with ID: $RUN_ID"
echo

echo "2. Testing JSON export (--skip-db mode)..."
./bin/lfst-checksum --skip-db --dir cmd/lfst-checksum > /dev/null
echo "   ✓ Checksum computation works"
echo

echo "3. Testing local mode (--local flag)..."
./bin/lfst-checksum --local --db test-remote.db --run-id "$RUN_ID" --step 1 --dir cmd/lfst-checksum
echo

echo "4. Testing JSON export/import manually..."
# Create test JSON with actual checksums
cat > test-remote.json <<'JSON'
{
  "run_id": 1,
  "step_number": 2,
  "checksums": [
    {
      "Path": "test1.txt",
      "CRC32": 123456,
      "SizeBytes": 100
    },
    {
      "Path": "test2.txt",
      "CRC32": 789012,
      "SizeBytes": 200
    }
  ],
  "computed_at": "2025-01-01T12:00:00Z"
}
JSON

echo "   Importing from JSON file..."
./bin/lfst-import --db test-remote.db test-remote.json
echo

echo "   Importing from stdin..."
cat test-remote.json | ./bin/lfst-import --db test-remote.db --stdin
echo

echo "5. Querying database..."
echo "   Checksums per step:"
./bin/lfst-query --db test-remote.db stats --run-id "$RUN_ID"
echo

echo "6. Testing forced remote mode (will fail if SSH not configured)..."
echo "   Note: This requires 'gojira' to be accessible and lfst-import installed there"
if ssh -o ConnectTimeout=2 -o BatchMode=yes gojira "which lfst-import" > /dev/null 2>&1; then
    echo "   gojira is accessible, testing remote mode..."
    # Note: This will actually send data to gojira if SSH is configured
    # ./bin/lfst-checksum --remote gojira --db /tmp/test-remote.db --run-id "$RUN_ID" --step 3 --dir cmd
    echo "   Skipping actual remote test to avoid modifying gojira"
else
    echo "   gojira not accessible or lfst-import not installed (expected for local testing)"
fi
echo

echo "7. Checking hostname detection..."
HOSTNAME=$(hostname)
echo "   Current hostname: $HOSTNAME"
if [ "$HOSTNAME" == "gojira" ]; then
    echo "   Running on gojira - would use local mode by default"
else
    echo "   NOT running on gojira - would use remote mode by default (if auto_remote enabled)"
fi
echo

echo "=== Remote Mode Test Complete ==="
echo "Test database: test-remote.db"
echo "Test JSON: test-remote.json"
