# Colima Deployment Strategy

> **Purpose**: Document containerization and deployment strategy for media server using Colima
> **Status**: Architecture validated - 0 broken pipes at 93 req/s (stress test 2025-12-11)
> **Last Updated**: 2025-12-11

---

## Executive Summary

The media server is now **containerization-ready** after eliminating the template serialization bottleneck (commit 21e1ef5). Load testing validates the architecture can handle 93 req/s with 10 concurrent workers without broken pipes.

**Colima Advantages:**
- Lightweight Docker/Kubernetes runtime for macOS
- No Docker Desktop license requirements
- Native macOS integration (APFS, extended attributes)
- Resource-efficient (vs Docker Desktop overhead)
- Built on Lima (Linux virtual machines on macOS)

---

## Prerequisites

### 1. Install Colima

```bash
# Install via Homebrew
brew install colima docker docker-compose

# Start Colima (default: 2 CPUs, 2GB RAM, 60GB disk)
colima start

# Or start with custom resources
colima start --cpu 4 --memory 8 --disk 100

# Verify installation
docker ps
colima status
```

### 2. Verify Environment

```bash
# Check Docker context
docker context use colima

# Verify Colima VM is running
colima list

# Check available resources
colima status
```

---

## Architecture Considerations

### Critical Success Factors

✅ **Template Serialization Fixed** (commit 21e1ef5)
- Server sends empty `allFilePaths` array
- Client fetches from `/api/filelist` on-demand
- No more 100k path serialization blocking requests

✅ **Load Testing Validated**
- 10 workers, 93 req/s sustained throughput
- 0 broken pipes under stress
- Response times: 3.76ms avg, 8.63ms p99

⚠️ **macOS-Specific Dependencies**
- Extended attributes (`com.apple.metadata:_kMDItemUserTags`)
- Finder comments (stored in extended attributes)
- APFS performance characteristics

⚠️ **APFS Cache Thrashing**
- APFS has known performance issues with deep directory hierarchies
- Limit scan depth to reduce cache thrashing
- Consider file organization strategy (shallower hierarchies)

---

## Deployment Strategy

### Option 1: Single Container with Volume Mounts (Recommended for Development)

**Architecture:**
- One container per media library
- Volume mount for media files (read-only)
- Volume mount for SQLite cache (persistent)
- Port mapping to host

**Advantages:**
- Simple setup
- Direct access to host filesystem
- Extended attributes preserved (if volume supports)
- Easy debugging

**Limitations:**
- No horizontal scaling
- Single point of failure
- Resource contention on single container

### Option 2: Multi-Container with Shared Storage (Production)

**Architecture:**
- Multiple container instances
- Shared volume for media files (NFS/SMB)
- Individual SQLite caches per container
- Load balancer (nginx/traefik) in front

**Advantages:**
- Horizontal scaling
- High availability
- Load distribution
- Rolling updates possible

**Limitations:**
- More complex setup
- Shared storage dependency
- Extended attribute support depends on filesystem

### Option 3: Kubernetes on Colima (Advanced)

**Architecture:**
- Colima with Kubernetes runtime
- Deployment with multiple replicas
- Persistent volume claims for cache
- Service with load balancing
- Ingress for external access

**Advantages:**
- Full orchestration capabilities
- Auto-scaling potential
- Health checks and self-healing
- Production-grade deployment model

**Limitations:**
- Complexity overhead
- Resource requirements
- Learning curve

---

## Dockerfile Strategy

### Production Dockerfile

```dockerfile
# Multi-stage build for minimal image size
FROM golang:1.21-alpine AS builder

WORKDIR /build

# Copy go mod files
COPY go.mod go.sum ./
RUN go mod download

# Copy source code
COPY . .

# Build static binary
RUN CGO_ENABLED=1 GOOS=linux go build -a -installsuffix cgo \
    -ldflags="-w -s" \
    -o media-server cmd/media-server/main.go

# Final stage - minimal runtime
FROM alpine:latest

# Install runtime dependencies
RUN apk --no-cache add ca-certificates sqlite-libs

# Create non-root user
RUN addgroup -g 1000 mediaserver && \
    adduser -D -u 1000 -G mediaserver mediaserver

WORKDIR /app

# Copy binary from builder
COPY --from=builder /build/media-server .

# Create directories for cache and media
RUN mkdir -p /data/cache /data/media && \
    chown -R mediaserver:mediaserver /data

USER mediaserver

# Expose port
EXPOSE 8080

# Health check
HEALTHCHECK --interval=30s --timeout=3s --start-period=5s --retries=3 \
    CMD wget --no-verbose --tries=1 --spider http://localhost:8080/ || exit 1

# Default command
CMD ["./media-server", "--dir=/data/media", "--port=8080"]
```

### Build and Tag

```bash
# Build image
docker build -t media-server:latest .

# Tag for versioning
docker tag media-server:latest media-server:v1.0.0

# Verify image
docker images | grep media-server
```

---

## Volume Strategy

### 1. Media Files (Read-Only)

**Option A: Direct Mount (Development)**
```bash
docker run -d \
  --name media-server \
  -v /Volumes/External/media:/data/media:ro \
  -p 8080:8080 \
  media-server:latest
```

**Option B: Named Volume (Production)**
```bash
# Create volume
docker volume create media-storage

# Copy data (one-time)
docker run --rm -v /Volumes/External/media:/source:ro \
  -v media-storage:/dest \
  alpine cp -r /source/. /dest/

# Run with volume
docker run -d \
  --name media-server \
  -v media-storage:/data/media:ro \
  -p 8080:8080 \
  media-server:latest
```

### 2. SQLite Cache (Persistent)

```bash
# Create cache volume
docker volume create media-cache

# Run with cache persistence
docker run -d \
  --name media-server \
  -v media-storage:/data/media:ro \
  -v media-cache:/data/cache \
  -p 8080:8080 \
  media-server:latest \
  --dir=/data/media --cache-dir=/data/cache
```

### 3. Extended Attributes Considerations

**macOS Extended Attributes in Containers:**

⚠️ **Critical**: Extended attributes may not work properly in Docker volumes on macOS!

**Workarounds:**

1. **Use NFS volumes with xattr support:**
```bash
# Mount with extended attribute support
docker run -d \
  --mount type=volume,src=media-storage,dst=/data/media,volume-driver=local,volume-opt=type=nfs,volume-opt=device=:/path/to/media,volume-opt=o=addr=host.docker.internal,xattr
```

2. **Copy tags to SQLite cache:**
```go
// Store tags in SQLite instead of extended attributes
// Trade-off: Lose direct Finder integration, gain container portability
```

3. **Hybrid approach:**
- Host process manages extended attributes
- Container serves cached data
- Sync mechanism between host and container

---

## Docker Compose Configuration

### Single Instance

```yaml
version: '3.8'

services:
  media-server:
    image: media-server:latest
    container_name: media-server
    restart: unless-stopped
    ports:
      - "8080:8080"
    volumes:
      - /Volumes/External/media:/data/media:ro
      - media-cache:/data/cache
    environment:
      - MEDIA_DIR=/data/media
      - CACHE_DIR=/data/cache
      - PORT=8080
    healthcheck:
      test: ["CMD", "wget", "--spider", "-q", "http://localhost:8080/"]
      interval: 30s
      timeout: 3s
      retries: 3
      start_period: 10s

volumes:
  media-cache:
    driver: local
```

### Multi-Instance with Load Balancer

```yaml
version: '3.8'

services:
  nginx:
    image: nginx:alpine
    container_name: media-lb
    restart: unless-stopped
    ports:
      - "80:80"
    volumes:
      - ./nginx.conf:/etc/nginx/nginx.conf:ro
    depends_on:
      - media-server-1
      - media-server-2
      - media-server-3

  media-server-1:
    image: media-server:latest
    container_name: media-server-1
    restart: unless-stopped
    expose:
      - "8080"
    volumes:
      - media-storage:/data/media:ro
      - media-cache-1:/data/cache
    environment:
      - MEDIA_DIR=/data/media
      - CACHE_DIR=/data/cache
      - PORT=8080

  media-server-2:
    image: media-server:latest
    container_name: media-server-2
    restart: unless-stopped
    expose:
      - "8080"
    volumes:
      - media-storage:/data/media:ro
      - media-cache-2:/data/cache
    environment:
      - MEDIA_DIR=/data/media
      - CACHE_DIR=/data/cache
      - PORT=8080

  media-server-3:
    image: media-server:latest
    container_name: media-server-3
    restart: unless-stopped
    expose:
      - "8080"
    volumes:
      - media-storage:/data/media:ro
      - media-cache-3:/data/cache
    environment:
      - MEDIA_DIR=/data/media
      - CACHE_DIR=/data/cache
      - PORT=8080

volumes:
  media-storage:
    driver: local
  media-cache-1:
    driver: local
  media-cache-2:
    driver: local
  media-cache-3:
    driver: local
```

### Nginx Load Balancer Config

```nginx
# nginx.conf
events {
    worker_connections 1024;
}

http {
    upstream media_backend {
        least_conn;  # Use least-connections load balancing
        server media-server-1:8080 max_fails=3 fail_timeout=30s;
        server media-server-2:8080 max_fails=3 fail_timeout=30s;
        server media-server-3:8080 max_fails=3 fail_timeout=30s;
    }

    server {
        listen 80;

        location / {
            proxy_pass http://media_backend;
            proxy_set_header Host $host;
            proxy_set_header X-Real-IP $remote_addr;
            proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
            proxy_set_header X-Forwarded-Proto $scheme;

            # Timeouts
            proxy_connect_timeout 60s;
            proxy_send_timeout 60s;
            proxy_read_timeout 60s;

            # Health checks
            proxy_next_upstream error timeout http_502 http_503 http_504;
        }

        # Health check endpoint
        location /health {
            access_log off;
            return 200 "healthy\n";
            add_header Content-Type text/plain;
        }
    }
}
```

---

## Deployment Workflow

### Initial Deployment

```bash
# 1. Build image
docker build -t media-server:latest .

# 2. Create volumes
docker volume create media-cache

# 3. Start container
docker-compose up -d

# 4. Verify health
docker ps
docker logs media-server
curl http://localhost:8080/

# 5. Load test
python3 load_test.py --url http://localhost:8080 --workers 5 --duration 30
```

### Updates and Rollback

```bash
# Zero-downtime update with multiple instances:

# 1. Build new version
docker build -t media-server:v1.1.0 .

# 2. Update one instance at a time
docker-compose up -d --no-deps --scale media-server=2 media-server

# 3. Wait for health check
sleep 10

# 4. Scale up new version, down old version
docker-compose up -d --no-deps --scale media-server=3

# 5. Verify no errors
docker logs media-server-1 --tail 100

# 6. Rollback if needed
docker-compose down
docker-compose up -d --force-recreate
```

---

## Monitoring and Health Checks

### Health Check Endpoint

```bash
# Add to main.go
http.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
    w.WriteHeader(http.StatusOK)
    json.NewEncoder(w).Encode(map[string]interface{}{
        "status": "healthy",
        "uptime": time.Since(startTime).String(),
        "files":  state.GetFileCount(),
        "tags":   state.GetCategoryCount(),
    })
})
```

### Container Logs

```bash
# View logs
docker logs media-server

# Follow logs
docker logs -f media-server

# Last 100 lines
docker logs media-server --tail 100

# Logs since timestamp
docker logs media-server --since 2025-12-11T09:00:00
```

### Resource Usage

```bash
# Container stats
docker stats media-server

# Colima VM stats
colima status

# Disk usage
docker system df
docker volume ls
```

---

## Performance Tuning

### Colima VM Resources

```bash
# Stop Colima
colima stop

# Restart with more resources
colima start --cpu 8 --memory 16 --disk 200

# For production workloads
colima start --cpu 16 --memory 32 --disk 500 --vm-type vz
```

### Go Runtime Tuning

```bash
# Set GOMAXPROCS to match container CPUs
docker run -d \
  --cpus="4" \
  -e GOMAXPROCS=4 \
  media-server:latest
```

### SQLite Cache Optimization

```bash
# Mount cache on tmpfs for maximum performance
docker run -d \
  --tmpfs /tmp/cache:rw,size=2g \
  -e CACHE_DIR=/tmp/cache \
  media-server:latest
```

---

## Troubleshooting

### Extended Attributes Not Working

**Symptom**: Tags not persisting or not readable in container

**Solution**:
1. Check if volume driver supports xattr: `docker volume inspect media-storage`
2. Use host network mode: `docker run --network host`
3. Consider SQLite-backed tag storage instead of extended attributes

### APFS Performance Issues

**Symptom**: Slow scanning, high CPU, cache thrashing

**Solution**:
1. Limit scan depth: `--max-depth 2`
2. Exclude deep directories: `--exclude-dirs`
3. Use shallower file organization
4. Pre-populate SQLite cache before container deployment

### Memory Pressure

**Symptom**: Container OOM killed

**Solution**:
1. Increase container memory: `--memory 4g`
2. Increase Colima VM memory: `colima start --memory 16`
3. Enable swap: `--memory-swap 8g`
4. Optimize SQLite cache size

### Port Conflicts

**Symptom**: "Port already in use" error

**Solution**:
```bash
# Find process using port
lsof -i :8080

# Kill process
kill -9 <PID>

# Or use different port
docker run -p 9191:8080 media-server:latest
```

---

## Security Considerations

### 1. Non-Root User

```dockerfile
# Already implemented in Dockerfile
RUN adduser -D -u 1000 mediaserver
USER mediaserver
```

### 2. Read-Only Filesystem

```bash
docker run -d \
  --read-only \
  --tmpfs /tmp \
  media-server:latest
```

### 3. Resource Limits

```yaml
services:
  media-server:
    deploy:
      resources:
        limits:
          cpus: '4'
          memory: 4G
        reservations:
          cpus: '2'
          memory: 2G
```

### 4. Network Isolation

```bash
# Create isolated network
docker network create media-net

# Run containers in isolated network
docker run -d \
  --network media-net \
  media-server:latest
```

---

## Production Readiness Checklist

- [ ] Template serialization bottleneck eliminated (commit 21e1ef5)
- [ ] Load testing passed (0 broken pipes at 93 req/s)
- [ ] Dockerfile optimized (multi-stage build, non-root user)
- [ ] Health check endpoint implemented
- [ ] Volume strategy defined (media + cache)
- [ ] Extended attribute handling decided
- [ ] Backup strategy for SQLite cache
- [ ] Monitoring and logging configured
- [ ] Resource limits set appropriately
- [ ] Load balancer configured (if multi-instance)
- [ ] Rolling update procedure documented
- [ ] Rollback procedure tested
- [ ] Security hardening applied
- [ ] Performance tuning completed

---

## Next Steps

1. **Implement Health Check Endpoint**
   - Add `/health` route in main.go
   - Return JSON with status, uptime, file/tag counts

2. **Build Docker Image**
   - Create Dockerfile
   - Test build process
   - Verify image size and layers

3. **Test Single Container Deployment**
   - Deploy with docker-compose
   - Verify extended attributes work (or implement workaround)
   - Run load tests against container

4. **Test Multi-Instance Deployment**
   - Deploy 3 instances with nginx load balancer
   - Verify load distribution
   - Test failover scenarios

5. **Document Operations Procedures**
   - Update procedures
   - Backup/restore procedures
   - Scaling procedures
   - Incident response runbook

---

## Related Documentation

- `PROJECT_OVERVIEW.md` - Architecture and implementation details
- `NEXT_CYCLE_IMPROVEMENTS.md` - Containerization bottleneck analysis and resolution
- `load_test.py` / `load_test.pl` - Load testing validation scripts
- Git commit 21e1ef5 - API endpoint fix that eliminated serialization bottleneck
- Git commit 8ecbf32 / d105425 - Load testing infrastructure

---

*Last stress test: 2025-12-11 - 2,796 requests, 92.87 req/s, 0 broken pipes, 3.76ms avg response time*
