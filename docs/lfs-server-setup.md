# LFS Test Server Setup

This document describes how to configure lfs-test-server on gojira for automated testing without user authentication prompts.

## Overview

For automated testing, lfs-test-server must be configured with:

1. Admin interface enabled (to manage users)
2. A test user account created
3. Client credentials embedded in LFS URL

## Server Configuration

### 1. Start lfs-test-server with Admin Interface

Create a startup script at `/opt/lfs-test-server/start-lfs-server.sh`:

```bash
#!/bin/bash
# Start lfs-test-server with admin interface enabled

export LFS_CONTENTPATH=/opt/lfs-test-server
export LFS_ADMINUSER=admin
export LFS_ADMINPASS=admin123

cd /opt/lfs-test-server

# Kill any existing instance
pkill -f lfs-test-server

# Start server in background with verbose logging
nohup ~/go/bin/lfs-test-server -verbose -addr :8080 > lfs-server.log 2>&1 &

echo "LFS Test Server started on port 8080"
echo "Admin interface: http://gojira:8080/mgmt"
echo "Admin credentials: admin / admin123"
echo "Log file: /opt/lfs-test-server/lfs-server.log"
```

Make it executable:

```bash
chmod +x /opt/lfs-test-server/start-lfs-server.sh
```

### 2. Start the Server

On gojira:

```bash
/opt/lfs-test-server/start-lfs-server.sh
```

### 3. Create Test User

The test user account is required for all LFS operations. Create it once via the admin interface:

```bash
curl -u admin:admin123 -X POST \
  -d "name=testuser&password=testpass" \
  http://gojira:8080/mgmt/add
```

Verify user was created:
```bash
curl -s -u admin:admin123 http://gojira:8080/mgmt/users | grep testuser
```

## Client Configuration

For automated testing without credential prompts, embed credentials in the LFS URL in `.lfsconfig`:

```ini
[lfs]
	url = http://testuser:testpass@gojira:8080
```

This allows Git LFS to authenticate automatically without user interaction.

## Environment Variables

The following environment variables control lfs-test-server behavior:

- **LFS_CONTENTPATH**: Directory for LFS object storage (default: `./lfs-content`)
- **LFS_ADMINUSER**: Admin username for `/mgmt` interface (no default)
- **LFS_ADMINPASS**: Admin password for `/mgmt` interface (no default)
- **LFS_HOST**: Server listen address (default: `localhost:8080`)

## Security Note

The credentials above (`testuser`/`testpass`) are for **testing only** on internal networks. Do not use these credentials in production environments or expose the server to public networks.

## Verification

Test that the server accepts LFS operations:

```bash
# Create test repo
cd /tmp && rm -rf lfs-test && mkdir lfs-test && cd lfs-test
git init
git lfs install

# Configure LFS URL with credentials
git config -f .lfsconfig lfs.url "http://testuser:testpass@gojira:8080"

# Track and commit a test file
echo "test" > test.txt
git lfs track "*.txt"
git add .
git commit -m "test"

# Create bare repo and push
ssh gojira "rm -rf /tmp/test-bare.git && git init --bare /tmp/test-bare.git"
git remote add origin gojira:/tmp/test-bare.git
git push origin master

# Should complete without credential prompts
```

## Troubleshooting

### Server Logs

Monitor server activity:
```bash
ssh gojira "tail -f /opt/lfs-test-server/lfs-server.log"
```

### Check Server Status

```bash
ssh gojira "ps aux | grep lfs-test-server | grep -v grep"
curl -v http://gojira:8080/
```

### Reset Server

To start fresh with a new database:
```bash
ssh gojira "cd /opt/lfs-test-server && pkill lfs-test-server && mv lfs.db lfs.db.bak"
/opt/lfs-test-server/start-lfs-server.sh
```

Then recreate the test user as shown in step 3 above.

## Automated Test Integration

The `lfst-scenario` command automatically uses the embedded credentials URL for scenarios 6 and 7 (LFS Test Server scenarios). No manual credential configuration is needed when running these scenarios.
