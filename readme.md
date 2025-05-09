# Test-Monitor Command Tool

A command-line interface tool for testing and monitoring 5G Core Network System and Mobile State Simulation (RAN) through a client-server architecture.

## Features

- Interactive command-line interface
- Client-server architecture for remote control
- Hierarchical command structure for intuitive navigation
- Support for various network testing operations
- Will be updated for more.

## Installation

### Prerequisites

- Go 1.24 
- Git

### Setup

1. Clone the repository:
   ```sh
   git clone https://github.com/yourusername/remote-control.git
   cd remote-control
   ```

2. Install dependencies:
   ```sh
   go mod download
   ```

## Server Setup

To run the server, navigate to the project root directory and run:

```sh
go run cmd/server/main.go
```

## Run the Client CLI

To run the client CLI, navigate to the project root directory and run:

```sh
go run cmd/client/main.go
```

## How to Use

The CLI provides a hierarchical interface for interacting with network elements. You can type "help" at any time to see available commands.

### Basic Navigation

1. **Connect to server**:
   ```
   >>> connect http://localhost:8080
   ```

2. **Navigate contexts**:
   ```
   >>> use emulator
   emulator >>> 
   ```

3. **Go back to previous context**:
   ```
   emulator >>> back
   >>> 
   ```

4. **Exit the program**:
   ```
   >>> exit
   ```

### Command Structure

Commands follow this general pattern:
```
[context] >>> [command] [subcommand] [arguments] [--options]
```

### Common Commands

- `help`: Display available commands
- `clear`: Clear the screen
- `connect <server-url>`: Connect to a server
- `disconnect`: Disconnect from current server
- `use <context>`: Switch to a specific context
- `back`: Return to previous context
- `select <item>`: Select a specific item in the current context

### Context-Specific Commands

#### Emulator Context
- `list-devices`: List all available devices
- `add-device`: Add a new device to the emulator
- `remove-device`: Remove a device from the emulator

#### Device Context
- `start`: Start the selected device
- `stop`: Stop the selected device
- `restart`: Restart the selected device
- `status`: Get the status of the selected device
- `configure`: Configure device parameters

## Troubleshooting

### Common Issues

1. **Cannot connect to server**
   - Ensure server is running
   - Check network connectivity
   - Verify port is not blocked by firewall

2. **Command not found**
   - Use `help` to see available commands
   - Make sure you're in the correct context

3. **Operation timeout**
   - Server might be busy or unresponsive
   - Check server logs for errors

## Development

### Project Structure

```
remote-control/
├── client/         # Client implementation
├── server/         # Server implementation
├── common/         # Shared utilities
├── go.mod
├── go.sum
└── README.md
```





