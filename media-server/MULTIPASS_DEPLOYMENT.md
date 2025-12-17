# Multipass Deployment Strategy

> **Purpose**: Document VM-based deployment strategy for media server using Multipass
> **Status**: Architecture validated - 0 broken pipes at 93 req/s (stress test 2025-12-11)
> **Last Updated**: 2025-12-11
> **Companion Document**: `COLIMA_DEPLOYMENT.md` (container strategy)

---

## Executive Summary

Multipass provides lightweight Ubuntu VMs on macOS/Linux/Windows, offering an alternative deployment model to containerization. This approach trades container density for VM isolation and traditional Linux deployment patterns.

**Multipass Advantages:**
- Full Ubuntu environment (native systemd, full Linux tooling)
- True VM isolation (kernel-level)
- Easier extended attribute handling (native Linux filesystem)
- Traditional deployment model (binary + systemd service)
- Works on macOS, Linux, and Windows hosts

**vs Colima:**
- Colima: Lima VMs running Docker/Kubernetes (container orchestration layer)
- Multipass: Ubuntu VMs running traditional Linux deployments (systemd, binaries)
- Both use VMs, but Colima abstracts VM management and presents container interface

---

## Prerequisites

### 1. Install Multipass

```bash
# Install via Homebrew (macOS)
brew install multipass

# Or download from: https://multipass.run/install

# Verify installation
multipass version
multipass list
```

### 2. Verify Environment

```bash
# Check Multipass is running
multipass list

# Check available images
multipass find

# Test VM creation
multipass launch --name test-vm
multipass shell test-vm
exit
multipass delete test-vm
multipass purge
```

---

## Architecture Considerations

### Deployment Models

#### Option 1: Single VM (Development/Small Scale)

**Architecture:**
- One Ubuntu VM running media-server binary
- Mounted volumes from host for media files
- SQLite cache on VM disk
- Systemd service for process management

**Advantages:**
- Simple setup
- Low resource overhead
- Easy debugging
- Direct systemd integration

**Limitations:**
- Single point of failure
- No horizontal scaling
- Manual load balancing if multiple VMs

#### Option 2: Multi-VM with Load Balancer (Production)

**Architecture:**
- Multiple Ubuntu VMs each running media-server
- Separate VM running nginx/HAProxy load balancer
- Shared NFS mount for media files
- Individual SQLite caches per VM

**Advantages:**
- High availability
- Horizontal scaling
- Load distribution
- Rolling updates

**Limitations:**
- More resource intensive (full VMs)
- NFS dependency for shared storage
- More complex management

---

## VM Specification

### Development VM

```bash
multipass launch \
  --name media-server-dev \
  --cpus 2 \
  --memory 2G \
  --disk 20G \
  22.04  # Ubuntu 22.04 LTS
```

### Production VM

```bash
multipass launch \
  --name media-server-prod \
  --cpus 4 \
  --memory 8G \
  --disk 50G \
  22.04
```

### Load Balancer VM

```bash
multipass launch \
  --name media-lb \
  --cpus 2 \
  --memory 1G \
  --disk 10G \
  22.04
```

---

## Volume Mounting Strategy

### Media Files (Read-Only)

```bash
# Mount host directory into VM
multipass mount /Volumes/External/media media-server-dev:/mnt/media

# Verify mount
multipass exec media-server-dev -- ls -la /mnt/media

# Make read-only (inside VM)
multipass exec media-server-dev -- sudo mount -o remount,ro /mnt/media
```

### SQLite Cache (Persistent on VM)

```bash
# Cache stored on VM disk at /var/lib/media-server/cache
# Persists across restarts
# Backed up with VM snapshots
```

---

## Installation Methods

### Method 1: Binary Deployment (Recommended)

```bash
# Build binary on host
GOOS=linux GOARCH=amd64 go build -o media-server-linux cmd/media-server/main.go

# Transfer to VM
multipass transfer media-server-linux media-server-dev:/tmp/

# Install on VM
multipass exec media-server-dev -- sudo mkdir -p /opt/media-server
multipass exec media-server-dev -- sudo mv /tmp/media-server-linux /opt/media-server/media-server
multipass exec media-server-dev -- sudo chmod +x /opt/media-server/media-server
```

### Method 2: Build on VM

```bash
# Install Go on VM
multipass exec media-server-dev -- bash <<'EOF'
  sudo apt update
  sudo apt install -y wget
  wget https://go.dev/dl/go1.21.5.linux-amd64.tar.gz
  sudo tar -C /usr/local -xzf go1.21.5.linux-amd64.tar.gz
  echo 'export PATH=$PATH:/usr/local/go/bin' >> ~/.bashrc
EOF

# Transfer source code
multipass transfer . media-server-dev:/tmp/media-server/

# Build on VM
multipass exec media-server-dev -- bash <<'EOF'
  cd /tmp/media-server
  /usr/local/go/bin/go build -o media-server cmd/media-server/main.go
  sudo mkdir -p /opt/media-server
  sudo mv media-server /opt/media-server/
  sudo chmod +x /opt/media-server/media-server
EOF
```

---

## Systemd Service Configuration

### Service File: `/etc/systemd/system/media-server.service`

```ini
[Unit]
Description=Media Server
After=network.target

[Service]
Type=simple
User=media-server
Group=media-server
WorkingDirectory=/opt/media-server
ExecStart=/opt/media-server/media-server \
  --dir=/mnt/media \
  --port=8080 \
  --cache-dir=/var/lib/media-server/cache
Restart=always
RestartSec=10
StandardOutput=journal
StandardError=journal
SyslogIdentifier=media-server

# Resource limits
LimitNOFILE=65536
MemoryLimit=4G

# Security hardening
NoNewPrivileges=true
PrivateTmp=true
ProtectSystem=strict
ProtectHome=true
ReadWritePaths=/var/lib/media-server/cache
ReadOnlyPaths=/mnt/media

[Install]
WantedBy=multi-user.target
```

### Installation

```bash
# Create service user
multipass exec media-server-dev -- sudo useradd -r -s /bin/false media-server

# Create cache directory
multipass exec media-server-dev -- sudo mkdir -p /var/lib/media-server/cache
multipass exec media-server-dev -- sudo chown media-server:media-server /var/lib/media-server/cache

# Install service file
multipass transfer media-server.service media-server-dev:/tmp/
multipass exec media-server-dev -- sudo mv /tmp/media-server.service /etc/systemd/system/

# Enable and start service
multipass exec media-server-dev -- sudo systemctl daemon-reload
multipass exec media-server-dev -- sudo systemctl enable media-server
multipass exec media-server-dev -- sudo systemctl start media-server

# Check status
multipass exec media-server-dev -- sudo systemctl status media-server
```

---

## Multi-VM Deployment

### VM Setup

```bash
# Launch 3 media server VMs
for i in 1 2 3; do
  multipass launch \
    --name media-server-$i \
    --cpus 4 \
    --memory 4G \
    --disk 30G \
    22.04

  # Mount media files
  multipass mount /Volumes/External/media media-server-$i:/mnt/media
done

# Launch load balancer VM
multipass launch \
  --name media-lb \
  --cpus 2 \
  --memory 2G \
  --disk 10G \
  22.04
```

### Load Balancer Configuration

**Install nginx:**

```bash
multipass exec media-lb -- sudo apt update
multipass exec media-lb -- sudo apt install -y nginx

# Transfer nginx config
multipass transfer nginx-lb.conf media-lb:/tmp/
multipass exec media-lb -- sudo mv /tmp/nginx-lb.conf /etc/nginx/sites-available/media-lb
multipass exec media-lb -- sudo ln -s /etc/nginx/sites-available/media-lb /etc/nginx/sites-enabled/
multipass exec media-lb -- sudo rm /etc/nginx/sites-enabled/default

# Restart nginx
multipass exec media-lb -- sudo systemctl restart nginx
```

**Nginx Configuration: `nginx-lb.conf`**

```nginx
upstream media_backend {
    least_conn;

    # Get VM IPs with: multipass list
    server <media-server-1-ip>:8080 max_fails=3 fail_timeout=30s;
    server <media-server-2-ip>:8080 max_fails=3 fail_timeout=30s;
    server <media-server-3-ip>:8080 max_fails=3 fail_timeout=30s;
}

server {
    listen 80;
    server_name _;

    location / {
        proxy_pass http://media_backend;
        proxy_set_header Host $host;
        proxy_set_header X-Real-IP $remote_addr;
        proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
        proxy_set_header X-Forwarded-Proto $scheme;

        proxy_connect_timeout 60s;
        proxy_send_timeout 60s;
        proxy_read_timeout 60s;

        proxy_next_upstream error timeout http_502 http_503 http_504;
    }

    location /health {
        access_log off;
        return 200 "healthy\n";
        add_header Content-Type text/plain;
    }
}
```

---

## Network Configuration

### Port Forwarding

```bash
# Forward VM port to host
# Not directly supported in Multipass - use SSH tunneling

# SSH tunnel from host to VM
ssh -L 8080:localhost:8080 ubuntu@$(multipass info media-server-dev --format json | jq -r '.info["media-server-dev"].ipv4[0]')
```

### VM-to-VM Communication

```bash
# VMs can communicate directly via their IP addresses
# Get VM IP:
multipass info media-server-1 --format json | jq -r '.info["media-server-1"].ipv4[0]'

# VMs are on same network by default
# Test connectivity:
multipass exec media-server-1 -- ping $(multipass info media-server-2 --format json | jq -r '.info["media-server-2"].ipv4[0]')
```

---

## Ansible Automation

See `ansible/` directory for deployment automation that works for both Multipass and Colima.

**Key Playbooks:**
- `ansible/site.yml` - Main deployment orchestration
- `ansible/roles/media-server/` - Application deployment
- `ansible/roles/load-balancer/` - Nginx setup
- `ansible/inventory/multipass.yml` - Multipass inventory
- `ansible/inventory/colima.yml` - Colima inventory

**Common tasks:**
- Binary deployment
- Service configuration
- Health check setup
- Monitoring configuration

**Platform-specific:**
- Multipass: systemd service, VM networking
- Colima: Docker containers, container networking

---

## Operations

### Starting VMs

```bash
# Start all VMs
multipass start --all

# Start specific VM
multipass start media-server-1
```

### Stopping VMs

```bash
# Stop all VMs
multipass stop --all

# Graceful stop with service shutdown
multipass exec media-server-1 -- sudo systemctl stop media-server
multipass stop media-server-1
```

### Service Management

```bash
# View logs
multipass exec media-server-1 -- sudo journalctl -u media-server -f

# Restart service
multipass exec media-server-1 -- sudo systemctl restart media-server

# Check status
multipass exec media-server-1 -- sudo systemctl status media-server
```

### Updates

```bash
# Build new binary
GOOS=linux GOARCH=amd64 go build -o media-server-linux cmd/media-server/main.go

# Deploy to VM
multipass transfer media-server-linux media-server-1:/tmp/
multipass exec media-server-1 -- sudo systemctl stop media-server
multipass exec media-server-1 -- sudo mv /tmp/media-server-linux /opt/media-server/media-server
multipass exec media-server-1 -- sudo chmod +x /opt/media-server/media-server
multipass exec media-server-1 -- sudo systemctl start media-server
```

### Snapshots

```bash
# Create snapshot before updates
multipass snapshot media-server-1 --name pre-update-$(date +%Y%m%d)

# List snapshots
multipass list --snapshots

# Restore snapshot
multipass restore media-server-1.pre-update-20251211
```

---

## Monitoring

### Health Checks

```bash
# Check service status
multipass exec media-server-1 -- sudo systemctl is-active media-server

# HTTP health check
curl http://$(multipass info media-server-1 --format json | jq -r '.info["media-server-1"].ipv4[0]'):8080/health
```

### Resource Usage

```bash
# VM resource usage
multipass info media-server-1

# Inside VM
multipass exec media-server-1 -- top
multipass exec media-server-1 -- free -h
multipass exec media-server-1 -- df -h
```

### Logs

```bash
# Systemd journal
multipass exec media-server-1 -- sudo journalctl -u media-server --since today

# Follow logs
multipass exec media-server-1 -- sudo journalctl -u media-server -f
```

---

## Troubleshooting

### VM Won't Start

```bash
# Check hypervisor status
multipass version
multipass list

# Restart Multipass daemon (macOS)
sudo launchctl stop com.canonical.multipassd
sudo launchctl start com.canonical.multipassd

# Check logs
multipass list --format json
```

### Mount Not Working

```bash
# Unmount and remount
multipass unmount media-server-1:/mnt/media
multipass mount /Volumes/External/media media-server-1:/mnt/media

# Check mount inside VM
multipass exec media-server-1 -- mount | grep /mnt/media
```

### Service Won't Start

```bash
# Check service logs
multipass exec media-server-1 -- sudo journalctl -u media-server -n 50

# Check binary
multipass exec media-server-1 -- ls -la /opt/media-server/media-server
multipass exec media-server-1 -- /opt/media-server/media-server --help

# Check permissions
multipass exec media-server-1 -- sudo -u media-server /opt/media-server/media-server --dir=/mnt/media --port=8080
```

---

## Comparison: Multipass vs Colima

| Aspect | Multipass | Colima |
|--------|-----------|--------|
| **VM Technology** | Hypervisor VMs (Ubuntu) | Lima VMs (containerd/Docker) |
| **Abstraction Layer** | Direct VM access | Container orchestration on VM |
| **Deployment Model** | Traditional (binary + systemd) | Modern (Docker images) |
| **Process Management** | systemd in VM | Container runtime |
| **Resource Model** | Per-VM allocation | Per-container allocation |
| **Density** | Lower (full Ubuntu per VM) | Higher (shared Lima VM, many containers) |
| **Ecosystem** | Linux tooling, apt/snap | Docker ecosystem, image registry |
| **Extended Attributes** | Native Linux filesystem support | Depends on volume driver |
| **Networking** | VM bridge networking | Container networking (bridge/host) |
| **Updates** | Binary replacement + systemctl restart | Image rebuild + container recreate |
| **Snapshots** | Full VM state snapshots | Container volumes + image layers |
| **Learning Curve** | Traditional sysadmin | Container/Kubernetes concepts |
| **Best For** | Traditional Linux ops, strong VM isolation | Cloud-native workflows, high density |

---

## Production Readiness Checklist

- [ ] VMs sized appropriately (CPU, memory, disk)
- [ ] Media files mounted correctly
- [ ] SQLite cache directory configured
- [ ] Systemd service configured and tested
- [ ] Service auto-restart enabled
- [ ] Resource limits set in systemd
- [ ] Security hardening applied (user, permissions)
- [ ] Log rotation configured
- [ ] Health check endpoint working
- [ ] Monitoring configured
- [ ] Backup strategy for SQLite cache
- [ ] Snapshot schedule defined
- [ ] Update procedure documented
- [ ] Rollback procedure tested
- [ ] Load balancer configured (if multi-VM)

---

## Related Documentation

- `COLIMA_DEPLOYMENT.md` - Container deployment strategy
- `ansible/` - Deployment automation
- `PROJECT_OVERVIEW.md` - Architecture overview
- `NEXT_CYCLE_IMPROVEMENTS.md` - Performance validation

---

*Last stress test: 2025-12-11 - 2,796 requests, 92.87 req/s, 0 broken pipes, 3.76ms avg response time*
