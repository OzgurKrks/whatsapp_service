.PHONY: help build up down logs clean restart rebuild

# Default target
help:
	@echo "Available commands:"
	@echo "  build    - Build Docker images"
	@echo "  up       - Start all services"
	@echo "  down     - Stop all services"
	@echo "  logs     - Show logs for all services"
	@echo "  clean    - Remove containers, networks, and volumes"
	@echo "  restart  - Restart all services"
	@echo "  rebuild  - Rebuild and restart all services"

# Build Docker images
build:
	docker-compose build

# Start all services
up:
	docker-compose up -d

# Stop all services
down:
	docker-compose down

# Show logs
logs:
	docker-compose logs -f

# Clean up everything
clean:
	docker-compose down -v --remove-orphans
	docker system prune -f

# Restart services
restart:
	docker-compose restart

# Rebuild and restart
rebuild: build up

# Show service status
status:
	docker-compose ps

# Access app container
shell:
	docker-compose exec boilerplate-app sh

# Access database
db:
	docker-compose exec boilerplate-db psql -U boilerplate_user -d boilerplate_db