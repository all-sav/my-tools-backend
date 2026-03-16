# ==============================================================================
# MyTools Makefile
# ==============================================================================

# Использование .ONESHELL для корректной обработки многострочных команд
.ONESHELL:

# Цвета для форматирования вывода
YELLOW := \033[1;33m
GREEN  := \033[1;32m
RED    := \033[1;31m
BLUE   := \033[1;34m
NC     := \033[0m # Без цвета

REDIS_CONTAINER := mt-redis-server

# Список всех доступных команд для .PHONY
.PHONY: env-create \
        redis-start redis-stop redis-rm redis-restart redis-cli redis-logs redis-status redis-flush

# ==============================================================================
# ОСНОВНЫЕ КОМАНДЫ
# ==============================================================================

# Копирование env.example в .env (принудительное пересоздание)
env-create:
	echo "$(BLUE)📋 Создание/пересоздание файла .env из env.example...$(NC)\n"
	cp -f .env.example .env
	echo "$(GREEN)✅ Файл .env создан/обновлен$(NC)\n"

# ==============================================================================
# УПРАВЛЕНИЕ REDIS
# ==============================================================================

# Запуск redis
redis-start:
	@echo "$(BLUE)Starting Redis container...$(NC)"
	@if docker ps -a | grep -q ${REDIS_CONTAINER}; then \
		if docker ps | grep -q ${REDIS_CONTAINER}; then \
			echo "$(GREEN)Redis already running$(NC)"; \
			docker ps | grep ${REDIS_CONTAINER}; \
		else \
			echo "$(YELLOW)Starting stopped Redis...$(NC)"; \
			docker start ${REDIS_CONTAINER}; \
			echo "$(GREEN)Redis started$(NC)"; \
			docker ps | grep ${REDIS_CONTAINER}; \
		fi \
	else \
		echo "$(GREEN)Creating new Redis container...$(NC)"; \
		if [ -f .env ]; then \
			REDIS_HOST=$$(grep REDIS_ADDR .env | cut -d= -f2); \
			REDIS_PORT=$$(grep REDIS_PORT .env | cut -d= -f2); \
			REDIS_PASS=$$(grep REDIS_PASSWORD .env | cut -d= -f2); \
			REDIS_DB=$$(grep REDIS_DB .env | cut -d= -f2); \
			PORT=$${REDIS_PORT:-6379}; \
			echo "$(YELLOW)Host: $$REDIS_HOST, Port: $$PORT, DB: $${REDIS_DB:-0}$(NC)"; \
			if [ -n "$$REDIS_PASS" ]; then \
				docker run -d --name ${REDIS_CONTAINER} -p $$PORT:6379 redis ${REDIS_CONTAINER} --requirepass $$REDIS_PASS; \
			else \
				docker run -d --name ${REDIS_CONTAINER} -p $$PORT:6379 redis; \
			fi \
		else \
			echo "$(YELLOW).env not found, using port 6379$(NC)"; \
			docker run -d --name ${REDIS_CONTAINER} -p 6379:6379 redis; \
		fi; \
		echo "$(GREEN)Redis created and started$(NC)"; \
		docker ps | grep ${REDIS_CONTAINER}; \
	fi

# Остановка Redis
redis-stop:
	@echo "$(BLUE)Stopping Redis container...$(NC)"
	@if docker ps | grep -q ${REDIS_CONTAINER}; then \
		docker stop ${REDIS_CONTAINER}; \
		echo "$(GREEN)Redis stopped$(NC)"; \
	else \
		echo "$(YELLOW)Redis container not running$(NC)"; \
	fi

# Проверка статуса Redis
redis-status:
	@echo "$(BLUE)Redis container status:$(NC)"
	@if docker ps -a | grep -q ${REDIS_CONTAINER}; then \
		echo "$(YELLOW)Container info:$(NC)"; \
		docker ps -a | grep --color=always ${REDIS_CONTAINER} || true; \
		echo ""; \
		if docker ps | grep -q ${REDIS_CONTAINER}; then \
			echo "$(YELLOW)Port mapping:$(NC)"; \
			docker port ${REDIS_CONTAINER}; \
			echo ""; \
		fi; \
		if [ -f .env ]; then \
			REDIS_HOST=$$(grep REDIS_ADDR .env | cut -d= -f2); \
			REDIS_PORT=$$(grep REDIS_PORT .env | cut -d= -f2); \
			REDIS_PASS=$$(grep REDIS_PASSWORD .env | cut -d= -f2); \
			REDIS_DB=$$(grep REDIS_DB .env | cut -d= -f2); \
			echo "$(YELLOW)Config from .env:$(NC)"; \
			echo "   REDIS_ADDR: $$REDIS_HOST"; \
			echo "   REDIS_PORT: $$REDIS_PORT"; \
			echo "   REDIS_DB: $${REDIS_DB:-0}"; \
			if [ -n "$$REDIS_PASS" ]; then \
				echo "   REDIS_PASSWORD: set"; \
			else \
				echo "   REDIS_PASSWORD: not set"; \
			fi; \
			echo ""; \
			if docker ps | grep -q ${REDIS_CONTAINER}; then \
				echo "$(YELLOW)Connection test (inside container):$(NC)"; \
				if [ -z "$$REDIS_PASS" ]; then \
					docker exec ${REDIS_CONTAINER} redis-cli -p 6379 PING 2>/dev/null | grep -q PONG && \
						echo "$(GREEN)Redis PONG (port 6379 inside container)$(NC)" || \
						echo "$(RED)Redis no response$(NC)"; \
				else \
					docker exec ${REDIS_CONTAINER} redis-cli -p 6379 -a $$REDIS_PASS PING 2>/dev/null | grep -q PONG && \
						echo "$(GREEN)Redis PONG (port 6379 inside container)$(NC)" || \
						echo "$(RED)Redis no response (wrong password?)$(NC)"; \
				fi; \
				echo ""; \
				echo "$(YELLOW)Connection test (from host):$(NC)"; \
				if command -v redis-cli >/dev/null 2>&1; then \
					if [ -z "$$REDIS_PASS" ]; then \
						redis-cli -h $$REDIS_HOST -p $$REDIS_PORT PING 2>/dev/null | grep -q PONG && \
							echo "$(GREEN)Redis accessible from host$(NC)" || \
							echo "$(RED)Redis NOT accessible from host$(NC)"; \
					else \
						redis-cli -h $$REDIS_HOST -p $$REDIS_PORT -a $$REDIS_PASS PING 2>/dev/null | grep -q PONG && \
							echo "$(GREEN)Redis accessible from host$(NC)" || \
							echo "$(RED)Redis NOT accessible from host$(NC)"; \
					fi \
				else \
					echo "$(YELLOW)redis-cli not installed on host, skipping check$(NC)"; \
				fi \
			fi \
		else \
			echo "$(YELLOW).env file not found$(NC)"; \
		fi \
	else \
		echo "$(RED)Redis container not found. Run: make redis-start$(NC)"; \
	fi

# Удаление Redis
redis-rm:
	@echo "$(BLUE)Removing Redis container...$(NC)"
	@if docker ps -a | grep -q ${REDIS_CONTAINER}; then \
		docker stop ${REDIS_CONTAINER} 2>/dev/null || true; \
		docker rm ${REDIS_CONTAINER}; \
		echo "$(GREEN)Redis container removed$(NC)"; \
	else \
		echo "$(YELLOW)Redis container not found$(NC)"; \
	fi

# Перезапуск Redis
redis-restart: redis-rm redis-start
	@echo "$(GREEN)Redis restarted with current .env settings$(NC)"

# Подключение к Redis CLI
redis-cli:
	@echo "$(BLUE)Connecting to Redis CLI...$(NC)"
	@if docker ps | grep -q ${REDIS_CONTAINER}; then \
		if [ -f .env ]; then \
			REDIS_PASS=$$(grep REDIS_PASSWORD .env | cut -d= -f2); \
			if [ -n "$$REDIS_PASS" ]; then \
				echo "$(YELLOW)Connecting with password...$(NC)"; \
				docker exec -it ${REDIS_CONTAINER} redis-cli -a $$REDIS_PASS; \
			else \
				echo "$(YELLOW)Connecting without password...$(NC)"; \
				docker exec -it ${REDIS_CONTAINER} redis-cli; \
			fi \
		else \
			docker exec -it ${REDIS_CONTAINER} redis-cli; \
		fi \
	else \
		echo "$(RED)Redis container not running. Run: make redis-start$(NC)"; \
	fi

# Просмотр логов
redis-logs:
	@echo "$(BLUE)Viewing Redis logs...$(NC)"
	@if docker ps | grep -q ${REDIS_CONTAINER}; then \
		docker logs -f ${REDIS_CONTAINER}; \
	else \
		echo "$(RED)Redis container not running$(NC)"; \
	fi

# Очистка данных Redis
redis-flush:
	@echo "$(BLUE)Flushing all Redis data...$(NC)"
	@if docker ps | grep -q ${REDIS_CONTAINER}; then \
		if [ -f .env ]; then \
			REDIS_PORT=$$(grep REDIS_PORT .env | cut -d= -f2); \
			REDIS_PASS=$$(grep REDIS_PASSWORD .env | cut -d= -f2); \
			PORT=$${REDIS_PORT:-6379}; \
			if [ -n "$$REDIS_PASS" ]; then \
				docker exec ${REDIS_CONTAINER} redis-cli -p 6379 -a $$REDIS_PASS FLUSHALL; \
			else \
				docker exec ${REDIS_CONTAINER} redis-cli -p 6379 FLUSHALL; \
			fi \
		else \
			docker exec ${REDIS_CONTAINER} redis-cli FLUSHALL; \
		fi; \
		echo "$(GREEN)All Redis data flushed$(NC)"; \
	else \
		echo "$(RED)Redis container not running$(NC)"; \
	fi
