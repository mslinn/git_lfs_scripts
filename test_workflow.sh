#!/bin/bash
# Comprehensive workflow test demonstrating all lfst commands
# This simulates a typical Git LFS testing workflow

set -e

# Colors for output
GREEN='\033[0;32m'
BLUE='\033[0;34m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

echo "=== Git LFS Testing Workflow Demo ===="
echo

# Configuration
TEST_DB="workflow-test.db"
TEST_REPO="test-lfs-repo"

# Clean up from previous runs
echo -e "${BLUE}Cleaning up previous test runs...${NC}"
rm -rf "$TEST_REPO" "$TEST_DB"
echo

# Step 1: Create a test run
echo -e "${BLUE}1. Creating test run for scenario 1...${NC}"
./bin/lfst-run --db "$TEST_DB" create \
    --scenario 1 \
    --server lfs-test-server \
    --protocol http \
    --notes "Workflow demonstration test"

RUN_ID=1
echo -e "   ${GREEN}✓${NC} Created test run ID: $RUN_ID"
echo

# Step 2: Create a test repository with some files
echo -e "${BLUE}2. Creating test Git repository...${NC}"
mkdir -p "$TEST_REPO"
cd "$TEST_REPO"

# Initialize git repo
git init --quiet
git config user.name "Test User"
git config user.email "test@example.com"

# Create some test files
echo "This is file one" > file1.txt
echo "This is file two with more content" > file2.txt
mkdir -p subdir
echo "File in subdirectory" > subdir/file3.txt
echo "Binary-like data: $(date)" > data.bin

git add .
git commit -m "Initial commit" --quiet

cd ..
echo -e "   ${GREEN}✓${NC} Repository created with 4 files"
echo

# Step 3: Compute initial checksums (Step 1)
echo -e "${BLUE}3. Step 1: Computing initial checksums...${NC}"
./bin/lfst-checksum --local --db "$TEST_DB" --run-id "$RUN_ID" --step 1 --dir "$TEST_REPO"
echo -e "   ${GREEN}✓${NC} Step 1 checksums stored"
echo

# Step 4: View the checksums
echo -e "${BLUE}4. Viewing stored checksums...${NC}"
./bin/lfst-query --db "$TEST_DB" checksums --run-id "$RUN_ID" --step 1
echo

# Step 5: Modify a file and compute new checksums (Step 2)
echo -e "${BLUE}5. Step 2: Modifying file1.txt and recomputing...${NC}"
echo "Additional content" >> "$TEST_REPO/file1.txt"
./bin/lfst-checksum --local --db "$TEST_DB" --run-id "$RUN_ID" --step 2 --dir "$TEST_REPO"
echo -e "   ${GREEN}✓${NC} Step 2 checksums stored"
echo

# Step 6: Compare the two steps
echo -e "${BLUE}6. Comparing Step 1 vs Step 2...${NC}"
./bin/lfst-query --db "$TEST_DB" compare --run-id "$RUN_ID" --from 1 --to 2
echo

# Step 7: Add a new file (Step 3)
echo -e "${BLUE}7. Step 3: Adding new file...${NC}"
echo "Brand new file" > "$TEST_REPO/newfile.txt"
./bin/lfst-checksum --local --db "$TEST_DB" --run-id "$RUN_ID" --step 3 --dir "$TEST_REPO"
echo -e "   ${GREEN}✓${NC} Step 3 checksums stored"
echo

# Step 8: Compare Step 2 vs Step 3
echo -e "${BLUE}8. Comparing Step 2 vs Step 3...${NC}"
./bin/lfst-query --db "$TEST_DB" compare --run-id "$RUN_ID" --from 2 --to 3
echo

# Step 9: Delete a file (Step 4)
echo -e "${BLUE}9. Step 4: Deleting data.bin...${NC}"
rm "$TEST_REPO/data.bin"
./bin/lfst-checksum --local --db "$TEST_DB" --run-id "$RUN_ID" --step 4 --dir "$TEST_REPO"
echo -e "   ${GREEN}✓${NC} Step 4 checksums stored"
echo

# Step 10: Compare to see deletion
echo -e "${BLUE}10. Comparing Step 3 vs Step 4 (should show deletion)...${NC}"
./bin/lfst-query --db "$TEST_DB" compare --run-id "$RUN_ID" --from 3 --to 4
echo

# Step 11: View run statistics
echo -e "${BLUE}11. Viewing test run statistics...${NC}"
./bin/lfst-query --db "$TEST_DB" stats --run-id "$RUN_ID"
echo

# Step 12: List all test runs
echo -e "${BLUE}12. Listing all test runs...${NC}"
./bin/lfst-run --db "$TEST_DB" list
echo

# Step 13: Show detailed run information
echo -e "${BLUE}13. Showing detailed run information...${NC}"
./bin/lfst-run --db "$TEST_DB" show "$RUN_ID"
echo

# Step 14: Mark run as completed
echo -e "${BLUE}14. Marking test run as completed...${NC}"
./bin/lfst-run --db "$TEST_DB" complete "$RUN_ID" --notes "All workflow steps passed successfully"
echo

# Step 15: View final run status
echo -e "${BLUE}15. Final run status...${NC}"
./bin/lfst-run --db "$TEST_DB" show "$RUN_ID"
echo

# Step 16: Overall database statistics
echo -e "${BLUE}16. Overall database statistics...${NC}"
./bin/lfst-query --db "$TEST_DB" stats
echo

# Step 17: Demonstrate skip-db mode
echo -e "${BLUE}17. Testing skip-db mode (no database operations)...${NC}"
echo -e "   ${YELLOW}Computing checksums without storing:${NC}"
./bin/lfst-checksum --skip-db --dir "$TEST_REPO" | head -8
echo

# Summary
echo -e "${GREEN}=== Workflow Test Complete ===${NC}"
echo
echo "This workflow demonstrated:"
echo "  ✓ Creating a test run"
echo "  ✓ Computing and storing checksums across multiple steps"
echo "  ✓ Detecting file modifications (Step 1 → Step 2)"
echo "  ✓ Detecting new files (Step 2 → Step 3)"
echo "  ✓ Detecting deleted files (Step 3 → Step 4)"
echo "  ✓ Querying checksums and generating reports"
echo "  ✓ Comparing checksums between steps"
echo "  ✓ Viewing run statistics"
echo "  ✓ Managing test run lifecycle (create → complete)"
echo
echo "Test artifacts:"
echo "  - Database: $TEST_DB"
echo "  - Repository: $TEST_REPO"
echo
echo "Try these commands:"
echo "  ./bin/lfst-run --db $TEST_DB list"
echo "  ./bin/lfst-query --db $TEST_DB stats"
echo "  ./bin/lfst-query --db $TEST_DB checksums --run-id $RUN_ID --step 1"
echo
echo "To clean up:"
echo "  rm -rf $TEST_REPO $TEST_DB"
echo
