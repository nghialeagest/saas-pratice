.PHONY: help build-all-images docker-up docker-down docker-logs run-monolith

help:
	@echo "Lệnh hữu ích cho saas-base:"
	@echo "  make build-all-images  - Build Docker image cho từng module (user, note, task, tenant)"
	@echo "  make docker-up         - Khởi chạy toàn bộ hệ thống module + Gateway Nginx"
	@echo "  make docker-down       - Dừng toàn bộ hệ thống Docker"
	@echo "  make docker-logs       - Theo dõi log của tất cả các container"
	@echo "  make test-backend      - Chạy toàn bộ test của backend"

build-all-images:
	docker build --build-arg MODULE=user -t saas-user:latest .
	docker build --build-arg MODULE=note -t saas-note:latest .
	docker build --build-arg MODULE=task -t saas-task:latest .
	docker build --build-arg MODULE=tenant -t saas-tenant:latest .

docker-up:
	docker compose -f docker-compose.modules.yml up -d --build 

docker-down:
	docker compose -f docker-compose.modules.yml down

docker-logs:
	docker compose -f docker-compose.modules.yml logs -f

test-backend:
	go test ./...
