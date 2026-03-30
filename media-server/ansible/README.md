# Ansible Deployment Automation

Unified deployment automation for media server supporting both **Multipass** (traditional VMs) and **Colima** (containerized) platforms.

## Quick Start

### Deploy to Multipass

```bash
# Deploy to Multipass VMs
ansible-playbook -i inventories/multipass.yml site.yml

# Deploy specific component
ansible-playbook -i inventories/multipass.yml site.yml --tags=media-server
ansible-playbook -i inventories/multipass.yml site.yml --tags=load-balancer
```

### Deploy to Colima

```bash
# Deploy to Colima containers
ansible-playbook -i inventories/colima.yml site.yml

# Deploy with custom variables
ansible-playbook -i inventories/colima.yml site.yml -e "media_dir=/custom/path"
```

## Architecture

### Common Components

- Media server deployment (binary or container)
- Health check configuration
- Monitoring setup
- Load balancer configuration

### Platform-Specific

**Multipass:**
- systemd service management
- Binary deployment
- VM networking

**Colima:**
- Docker container management
- Image build and deployment
- Container networking

## Inventory Structure

```
inventories/
├── multipass.yml       # Multipass VM inventory
├── colima.yml          # Colima container inventory
└── common_vars.yml     # Shared variables
```

## Roles

### media-server

Deploys the media server application.

**Tasks:**
- Build/transfer binary (Multipass) or build image (Colima)
- Configure service (systemd or Docker)
- Set up health checks
- Configure logging

### load-balancer

Deploys nginx load balancer.

**Tasks:**
- Install nginx
- Configure upstream servers
- Set up health checks
- Enable service

## Variables

### Global Variables (group_vars/all.yml)

```yaml
media_dir: /mnt/media
cache_dir: /var/lib/media-server/cache
port: 8080
```

### Multipass-Specific

```yaml
deployment_type: multipass
binary_path: /opt/media-server/media-server
service_manager: systemd
```

### Colima-Specific

```yaml
deployment_type: colima
image_name: media-server
image_tag: latest
container_runtime: docker
```

## Usage Examples

### Initial Deployment

```bash
# Multipass
ansible-playbook -i inventories/multipass.yml site.yml

# Colima
ansible-playbook -i inventories/colima.yml site.yml
```

### Update Application

```bash
# Multipass (binary update)
ansible-playbook -i inventories/multipass.yml site.yml --tags=deploy --skip-tags=setup

# Colima (rebuild and redeploy image)
ansible-playbook -i inventories/colima.yml site.yml --tags=build,deploy
```

### Scale Deployment

```bash
# Add more instances (edit inventory first)
ansible-playbook -i inventories/multipass.yml site.yml --limit=new-hosts
```

## Requirements

```bash
# Install Ansible
pip3 install --break-system-packages ansible

# Verify installation
ansible --version
```

## Tags

- `setup` - Initial setup tasks
- `build` - Build binary or image
- `deploy` - Deployment tasks
- `media-server` - Media server only
- `load-balancer` - Load balancer only
- `health-check` - Health check configuration
- `monitoring` - Monitoring setup

## Testing

```bash
# Check connectivity
ansible -i inventories/multipass.yml all -m ping

# Run in check mode (dry-run)
ansible-playbook -i inventories/multipass.yml site.yml --check

# Verbose output
ansible-playbook -i inventories/multipass.yml site.yml -vvv
```

## Related Documentation

- `../MULTIPASS_DEPLOYMENT.md` - Multipass deployment strategy
- `../COLIMA_DEPLOYMENT.md` - Colima deployment strategy
- `../PROJECT_OVERVIEW.md` - Application architecture
