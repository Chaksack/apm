# APM Quick Start Guide

Get your APM system up and running in 15 minutes with this comprehensive quick start guide.

## Prerequisites

Before you begin, ensure you have the following installed:

### Required Software
- **Docker** (version 20.10+) and **Docker Compose** (version 2.0+)
- **Git** for cloning the repository
- **Go** (version 1.21+) for development
- **Node.js** (version 18+) and **npm** for frontend development

### System Requirements
- **Memory**: Minimum 8GB RAM (16GB recommended)
- **CPU**: 4 cores minimum
- **Storage**: 50GB free space
- **Network**: Stable internet connection for dependency downloads

### Quick Installation Check
```bash
# Verify installations
docker --version
docker-compose --version
go version
node --version
npm --version
```

## 1. Clone and Setup Repository

```bash
# Clone the repository
git clone https://github.com/your-org/apm.git
cd apm

# Verify project structure
ls -la
```

## 2. Environment Configuration

### Copy Environment Files
```bash
# Copy example environment files
cp .env.example .env
cp docker-compose.override.yml.example docker-compose.override.yml
```

### Configure Environment Variables
Edit `.env` file with your settings:

```bash
# Database Configuration
DB_HOST=localhost
DB_PORT=5432
DB_NAME=apm_db
DB_USER=apm_user
DB_PASSWORD=your_secure_password

# Redis Configuration
REDIS_HOST=localhost
REDIS_PORT=6379
REDIS_PASSWORD=your_redis_password

# Application Configuration
APP_ENV=development
APP_PORT=8080
API_BASE_URL=http://localhost:8080
JWT_SECRET=your_jwt_secret_key

# Monitoring Configuration
PROMETHEUS_URL=http://localhost:9090
GRAFANA_URL=http://localhost:3000
JAEGER_URL=http://localhost:16686

# Log Configuration
LOG_LEVEL=info
LOG_FORMAT=json
```

## 3. Local Development Setup

### Backend Setup
```bash
# Navigate to backend directory
cd backend

# Install Go dependencies
go mod download

# Run database migrations
go run cmd/migrate/main.go up

# Start the backend server
go run cmd/server/main.go
```

### Frontend Setup
```bash
# In a new terminal, navigate to frontend directory
cd frontend

# Install npm dependencies
npm install

# Start the development server
npm run dev
```

## 4. Docker Compose Quick Start

For the fastest setup, use Docker Compose:

### Start All Services
```bash
# Start all services in detached mode
docker-compose up -d

# View service status
docker-compose ps

# View logs
docker-compose logs -f
```

### Service Access Points
After startup, access the following services:

- **APM Dashboard**: http://localhost:3000
- **API Server**: http://localhost:8080
- **Prometheus**: http://localhost:9090
- **Grafana**: http://localhost:3001
- **Jaeger UI**: http://localhost:16686
- **PostgreSQL**: localhost:5432
- **Redis**: localhost:6379

### Default Credentials
- **Grafana**: admin/admin
- **Database**: apm_user/your_secure_password

## 5. Verify Installation

### Health Check Commands
```bash
# Check API health
curl http://localhost:8080/health

# Check database connection
curl http://localhost:8080/api/v1/health/db

# Check Redis connection
curl http://localhost:8080/api/v1/health/redis

# View running containers
docker-compose ps
```

### Load Sample Data
```bash
# Load sample application data
curl -X POST http://localhost:8080/api/v1/sample-data \
  -H "Content-Type: application/json"

# Generate sample metrics
go run scripts/generate-sample-metrics.go
```

## 6. Common Issues and Solutions

### Port Conflicts
If you encounter port conflicts:
```bash
# Check what's using a port
lsof -i :8080

# Stop conflicting services
sudo lsof -ti:8080 | xargs kill -9
```

### Database Connection Issues
```bash
# Reset database
docker-compose down -v
docker-compose up -d postgres
sleep 10
docker-compose up -d
```

### Memory Issues
```bash
# Check Docker memory usage
docker stats

# Increase Docker memory allocation (Docker Desktop)
# Go to Settings > Resources > Advanced > Memory
```

## 7. Development Workflow

### Hot Reload Setup
```bash
# Backend hot reload with air
go install github.com/cosmtrek/air@latest
air

# Frontend hot reload (already configured)
npm run dev
```

### Testing
```bash
# Run backend tests
go test ./...

# Run frontend tests
npm test

# Run integration tests
docker-compose -f docker-compose.test.yml up --abort-on-container-exit
```

## 8. Next Steps

After completing the quick start:

1. **Explore the Dashboard**: Navigate to http://localhost:3000
2. **Review Configuration**: Check `docs/deployment/` for production setup
3. **API Documentation**: Visit http://localhost:8080/swagger
4. **Monitor Metrics**: Open Grafana at http://localhost:3001
5. **Trace Requests**: Use Jaeger at http://localhost:16686

## 9. Useful Commands

### Docker Management
```bash
# View logs for specific service
docker-compose logs -f api

# Restart a service
docker-compose restart api

# Rebuild and restart
docker-compose up -d --build api

# Clean up everything
docker-compose down -v --rmi all
```

### Database Operations
```bash
# Access PostgreSQL shell
docker-compose exec postgres psql -U apm_user -d apm_db

# Run migrations
docker-compose exec api go run cmd/migrate/main.go up

# Backup database
docker-compose exec postgres pg_dump -U apm_user apm_db > backup.sql
```

### Monitoring Commands
```bash
# View resource usage
docker stats

# Check service health
docker-compose exec api curl localhost:8080/health

# View application logs
docker-compose logs -f --tail=100 api
```

## 10. Support and Resources

- **Documentation**: `docs/` directory
- **API Reference**: http://localhost:8080/swagger
- **Configuration**: `config/` directory
- **Scripts**: `scripts/` directory
- **Issues**: Check GitHub issues for common problems

## Troubleshooting Checklist

- [ ] All prerequisites installed and correct versions
- [ ] Docker daemon running
- [ ] Ports 3000, 8080, 9090, 5432, 6379 available
- [ ] Environment variables configured
- [ ] Database migrations completed
- [ ] All services started successfully
- [ ] Health checks passing

---

**Time to Complete**: ~15 minutes  
**Difficulty**: Beginner  
**Next**: See `kubernetes-deployment.md` for production deployment