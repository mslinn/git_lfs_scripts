#!/bin/bash
# Test script for Stage 1 components

set -e

echo "=== Scenario 6 Test Script ==="
echo

# Clean up any previous test database
rm -f test_scenario6.db

echo "1. Creating a test run..."
# This will automatically create the database schema
# Scenario 6 = LFS Test Server - HTTP (server/protocol derived from scenario)
../bin/lfst-run create \
  --db test_scenario6.db \
  --scenario 6 \
  --notes "Stage 1 test run"

# Get the run ID (it will be 1 since this is a fresh database)
RUN_ID=1
echo "   Created test run with ID: $RUN_ID"
echo

echo "2. Computing checksums for bin/ directory (step 1)..."
../bin/lfst-checksum \
  --db test_scenario6.db \
  --run-id "$RUN_ID" \
  --step 1 \
  --dir bin \
  --local
echo

echo "3. Computing checksums for pkg/ directory (step 2)..."
../bin/lfst-checksum --db test_scenario6.db --run-id "$RUN_ID" --step 2 --dir pkg --local
echo

echo "4. Comparing step 2 with step 1..."
../bin/lfst-query --db test_scenario6.db compare --run-id "$RUN_ID" --from 1 --to 2
echo

echo "5. Querying database for results..."
echo "   Test runs:"
../bin/lfst-run --db test_scenario6.db list
echo
echo "   Checksums for step 1:"
../bin/lfst-query --db test_scenario6.db checksums --run-id "$RUN_ID" --step 1 --limit 10
echo

echo "6. Viewing statistics..."
../bin/lfst-query --db test_scenario6.db stats --run-id "$RUN_ID"
echo

echo "7. Testing verbose mode..."
../bin/lfst-checksum -v --skip-db --dir cmd/lfst-checksum | head -10
echo

echo "8. Marking test run as completed..."
../bin/lfst-run --db test_scenario6.db complete "$RUN_ID" --notes "All Stage 1 tests passed"
echo

echo "=== Stage 1 Test Complete ==="
echo "Test database: test_scenario6.db"
echo
echo "To inspect the database with lfst commands:"
echo "  ../bin/lfst-run --db test_scenario6.db list"
echo "  ../bin/lfst-run --db test_scenario6.db show $RUN_ID"
echo "  ../bin/lfst-query --db test_scenario6.db stats"
