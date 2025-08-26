# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

This is a Go-based WeChat robot management API service that provides:
- WeChat robot configuration management
- User login management with QR code scanning
- Database persistence with MySQL and GORM
- RESTful API endpoints for CRUD operations
- WebSocket client for real-time message receiving (configured but not implemented)
- OCR processing for image messages (configured but not implemented)
- Message queuing with RabbitMQ (configured but not implemented)
- File storage with MinIO (configured but not implemented)

## Architecture

### Main Components
- **Modular architecture**: Code is organized into separate files:
  - `main.go` (~700 lines) - Application core, configuration, routing, API handlers
  - `models.go` (~65 lines) - Database models and API response structures  
  - `service.go` (~430 lines) - WeChat robot service interface and HTTP client implementation
- **Configuration-driven**: Uses TOML configuration with comprehensive settings in `config.toml`
- **Database layer**: GORM with MySQL for data persistence and auto-migration
- **Logging**: Structured logging with Zap and log rotation via lumberjack
- **HTTP server**: Gin-based REST API with comprehensive endpoints
- **SQL schema**: Complete database schema provided in `database.sql`

### Key Dependencies
- `github.com/gin-gonic/gin` - HTTP framework
- `gorm.io/gorm` + `gorm.io/driver/mysql` - Database ORM
- `go.uber.org/zap` - Structured logging  
- `github.com/spf13/viper` - Configuration management
- `github.com/gorilla/websocket` - WebSocket client
- `github.com/rabbitmq/amqp091-go` - Message queuing
- `github.com/minio/minio-go/v7` - Object storage

## Development Commands

### Build and Run
```bash
# Build the application
go build -o wx-msg-api main.go models.go service.go

# Run directly
go run main.go models.go service.go

# Run with specific config
go run main.go models.go service.go
```

### Testing
```bash
# Run all tests
go test ./...

# Run tests with verbose output
go test -v ./...

# Run specific test
go test -run TestName
```

### Dependencies
```bash
# Install/update dependencies
go mod tidy

# Add new dependency
go get package-name

# Verify dependencies
go mod verify
```

## Configuration

The application uses `config.toml` for all configuration. Key sections:
- `[app]` - Basic application settings
- `[server]` - HTTP server configuration  
- `[database]` - MySQL connection settings
- `[redis]` - Redis configuration (not yet implemented in main.go)
- `[rabbitmq]` - Message queue settings
- `[websocket]` - WebSocket client configuration
- `[webhook]` - Notification webhook settings
- `[minio]` - File storage configuration
- `[ocr]` - OCR service configuration

## Key Implementation Details

### Database Connection
- Uses GORM with MySQL driver
- Connection pooling configured via config
- Simple DSN construction: `username:password@tcp(host:port)/database?charset=utf8mb4&parseTime=true&loc=Local`

### Logging System
- Dual output: console and rotating log files
- Configurable log levels (debug, info, warn, error)
- JSON or console format options
- Automatic log rotation and compression

### HTTP Server
- Health check endpoint at `/health` with component status
- Graceful shutdown with 30-second timeout
- Configurable timeouts for read/write/idle

### Database Schema
- **wx_robot_configs**: Robot configuration (address, admin_key, owner_id)  
- **wx_user_logins**: User login sessions (token, wx_id, nick_name, status)
- **wx_groups**: WeChat group information (group_id, group_nick_name)

### API Endpoints

**Robot Management:**
- `GET /api/wx/v1/robots/` - List all robots with their users
- `POST /api/wx/v1/robots/` - Create new robot configuration
- `DELETE /api/wx/v1/robots/:id` - Delete robot configuration

**User Login Management:**
- `GET /api/wx/v1/users/robot/:robotId` - List users for specific robot
- `POST /api/wx/v1/users/authorize` - Get authorization token (calls GenAuthKey API)
- `POST /api/wx/v1/users/qrcode` - Generate QR code for login (calls GetLoginQrCode API)
- `GET /api/wx/v1/users/status/:robotId/:token` - Check login status (calls CheckLoginStatus API)
- `DELETE /api/wx/v1/users/:id` - Delete user login

**Group Management:**
- `GET /api/wx/v1/groups/robot/:robotId/:token` - Get group list (calls GroupList API)
- `POST /api/wx/v1/groups/info` - Get group details (calls GetChatRoomInfo API)
- `GET /api/wx/v1/groups/login-status/:robotId/:token` - Get online status (calls GetLoginStatus API)
- `GET /api/wx/v1/groups/init-status/:robotId/:token` - Get initialization status (calls GetInitStatus API)

**Authorization Management:**
- `POST /api/wx/v1/auth/extend/:robotId` - Extend authorization (calls DelayAuthKey API)

**System:**
- `GET /health` - System health check

### Service Architecture
- Full CRUD operations for robot and user management  
- Real WeChat robot API integration via HTTP service calls
- Complete login flow: authorize → QR code → status checking → user save
- Group management with list and detail operations
- Authorization extension and status monitoring
- Automatic database migration on startup
- Pure API service without frontend interface

### External API Integration
All external WeChat robot API calls are handled through the `WxRobotService` interface with HTTP client implementation. The service calls the following external endpoints:
- `/admin/GenAuthKey1` - Generate authorization keys
- `/login/GetLoginQrCodeNewX` - Get login QR codes
- `/login/CheckLoginStatus` - Check scanning/login status
- `/login/CheckCanSetAlias` - Check security risks
- `/login/GetLoginStatus` - Get online status
- `/login/GetInItStatus` - Check initialization status
- `/admin/DelayAuthKey` - Extend authorization
- `/group/GroupList` - Get group lists
- `/group/GetChatRoomInfo` - Get group details

## Development Notes

- **Modular Design**: Code is properly separated into logical modules:
  - `main.go`: Application bootstrap, configuration, routing, and API handlers
  - `models.go`: Database schema definitions and API response structures  
  - `service.go`: External API service interface and HTTP client implementations
- **Production Ready**: Full WeChat robot API integration with real HTTP service calls
- **Clean Architecture**: Clear separation of concerns between data models, business logic, and HTTP handlers  
- **Configuration Driven**: All settings externalized to TOML configuration file
- WebSocket client, RabbitMQ, and other integrations are configured but not yet implemented in the service layer