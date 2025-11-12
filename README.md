# fwkeeper

Port forwarding made easy. A Kubernetes port forwarding tool that automatically maintains persistent port forwards to pods with automatic reconnection and failure recovery.

## Features

- 🔄 **Automatic Port Forwarding**: Maintains persistent port forwards to Kubernetes pods
- 🔁 **Automatic Reconnection**: Automatically reconnects on pod restarts or connection failures
- ⚙️ **Easy Configuration**: CUE-based configuration for simple pod forwarding setup
- 📊 **Structured Logging**: Comprehensive logging with configurable levels and pretty-printing
- 🔐 **Kubernetes Integration**: Seamless integration with your Kubernetes cluster (local or in-cluster)
- 🚀 **Multiple Forwards**: Support for multiple simultaneous port forwards

## Installation

### Prerequisites

- Go 1.25.1 or later
- Access to a Kubernetes cluster (local kubeconfig or in-cluster)
- kubectl configured (for cluster access)

### Build from Source

```bash
git clone <repository-url>
cd fwkeeper
go build -o fwkeeper ./
```

The binary will be created as `fwkeeper` in the current directory.

## Quick Start

### 1. Create a Configuration File

Create a `fwkeeper.cue` file:

```cue
logs: {
  level: "debug"
  pretty: true
}

forwards: [
  {
    name: "postgres"
    ports: ["5432"]
    namespace: "default"
    resource: "postgres-pod"
  },
  {
    name: "api"
    ports: ["8080:8080", "9000:9000"]
    namespace: "api"
    resource: "api-server-deployment-abc123"
  }
]
```

### 2. Run fwkeeper

```bash
./fwkeeper run -c fwkeeper.cue
```

Or using the default `fwkeeper.cue`:

```bash
./fwkeeper run
```

### 3. Access Your Services

The port forwards are now active. Connect to your services:

```bash
# Connect to postgres
psql -h localhost -p 5432

# Connect to API
curl http://localhost:8080
```

## Configuration

### Configuration File Format (CUE)

fwkeeper uses [CUE](https://cuelang.org/) for configuration validation and schema enforcement.

#### Top-Level Structure

```cue
logs: { ... }
forwards: [ ... ]
```

#### Logs Configuration

```cue
logs: {
  level: "debug"      # "error", "warn", "info", "debug", "trace" (default: "info")
  pretty: true        # Pretty-print logs to console (default: false)
}
```

#### Port Forward Configuration

```cue
forwards: [
  {
    name: "service-name"              # Unique identifier for this forward
    ports: ["8080", "9000:3000"]      # Local:remote port mappings
    namespace: "default"              # Kubernetes namespace
    resource: "pod-name"              # Pod name to forward to
  },
  # ... more forwards
]
```

**Port Mapping Syntax:**
- `"8080"` - Forward local port 8080 to pod port 8080
- `"8080:9000"` - Forward local port 8080 to pod port 9000

**Validation Rules:**
- Port numbers must be between 1 and 65535
- Each forward must have a unique name
- Namespace and resource (pod name) must be specified

### Environment Variables

```bash
# Override kubeconfig location
KUBECONFIG=/path/to/kubeconfig ./fwkeeper run

# Use default kubeconfig locations
# 1. $KUBECONFIG environment variable
# 2. ~/.kube/config
# 3. In-cluster config (if running inside Kubernetes)
```

## Usage

### Basic Commands

```bash
# Show help
./fwkeeper --help

# Run with default config (fwkeeper.cue)
./fwkeeper run

# Run with custom config file
./fwkeeper run -c /path/to/config.cue

# Show command help
./fwkeeper run --help
```

### Exit and Shutdown

Press `Ctrl+C` to gracefully stop fwkeeper. It will:
1. Cancel all active port forwards
2. Close connections
3. Print shutdown message

## Examples

### Example 1: Database Access

Forward to a PostgreSQL database pod:

```cue
logs: {
  level: "info"
  pretty: true
}

forwards: [
  {
    name: "database"
    ports: ["5432"]
    namespace: "databases"
    resource: "postgres-primary"
  }
]
```

Then connect:
```bash
psql -h localhost -p 5432 -U myuser -d mydb
```

### Example 2: Development Environment

Forward multiple services for local development:

```cue
logs: {
  level: "debug"
  pretty: true
}

forwards: [
  {
    name: "api-server"
    ports: ["8000:8000"]
    namespace: "development"
    resource: "api-server"
  },
  {
    name: "frontend"
    ports: ["3000:3000"]
    namespace: "development"
    resource: "frontend"
  },
  {
    name: "postgres"
    ports: ["5432:5432"]
    namespace: "databases"
    resource: "postgres-dev"
  },
  {
    name: "redis"
    ports: ["6379:6379"]
    namespace: "databases"
    resource: "redis-dev"
  }
]
```

### Example 3: Multiple Ports on Same Pod

Forward multiple ports from a single pod:

```cue
forwards: [
  {
    name: "api-services"
    ports: ["8000:8000", "9000:9000", "5000:5000"]
    namespace: "api"
    resource: "api-pod"
  }
]
```

## Logging

### Log Levels

- **error**: Only errors
- **warn**: Warnings and errors
- **info**: General information (default)
- **debug**: Detailed debugging information
- **trace**: Very detailed tracing information

### Log Output

Logs are written to stderr. Each log entry includes:
- Timestamp (Unix milliseconds)
- Log level
- Message
- Error details (when applicable)

### Pretty Printing

Enable pretty-printed logs for development:

```cue
logs: {
  pretty: true
}
```

Output will include timestamps and color-coded levels for better readability.

## Troubleshooting

### "Pod not in running state"

The pod exists but isn't currently running. Check pod status:

```bash
kubectl get pods -n <namespace> <pod-name>
kubectl describe pod -n <namespace> <pod-name>
```

### "Connection refused" or "Connection reset"

The pod restarted or the port-forward connection dropped. fwkeeper will automatically reconnect after 1 second. Check logs for details.

### "Unable to connect to kubeconfig"

Verify your kubeconfig:

```bash
# Check KUBECONFIG env var
echo $KUBECONFIG

# Test cluster access
kubectl cluster-info

# Set correct kubeconfig
export KUBECONFIG=~/.kube/config
./fwkeeper run
```

### Configuration Validation Errors

Verify your CUE configuration syntax and against the schema:

```bash
# Check for CUE syntax errors in your config file
# Ensure all fields match the required structure in schema.cue
```

Common issues:
- Missing required fields (name, ports, namespace, resource)
- Invalid port numbers (not 1-65535)
- Malformed port specifications (should be "port" or "local:remote")

### Debug Logs

Enable debug logging to see detailed information:

```cue
logs: {
  level: "debug"
  pretty: true
}
```

## Development

### Project Structure

```
fwkeeper/
├── cmd/                    # CLI command definitions
│   ├── root.go            # Root command
│   └── run.go             # Run command
├── internal/
│   ├── bootstrap/         # Dependency injection setup
│   ├── config/            # Configuration loading and validation
│   │   └── schema.cue     # CUE schema definition
│   ├── logger/            # Logging setup
│   └── service/           # Core forwarding logic
│       ├── forward.go     # Port forwarder implementation
│       ├── runner.go      # Main orchestrator
│       ├── kubernetes.go  # Kubernetes client setup
│       └── locator.go     # Pod locator
├── main.go                # Application entry point
├── go.mod                 # Go module definition
├── go.sum                 # Dependency checksums
├── fwkeeper.cue           # Default configuration
└── README.md              # This file
```

### Building

```bash
# Build the binary
go build -o fwkeeper ./

# Build with version info
go build -ldflags="-X main.version=v1.0.0" -o fwkeeper ./
```

### Running Tests

```bash
# Run all tests
go test ./...

# Run with verbose output
go test -v ./...

# Run specific test
go test -run TestName ./path/to/package
```

### Code Quality

```bash
# Format code
go fmt ./...

# Run static analysis
go vet ./...

# Run golangci-lint (if installed)
golangci-lint run ./...
```

### Dependencies

Key dependencies:
- **[Cobra](https://github.com/spf13/cobra)**: CLI framework
- **[CUE](https://cuelang.org/)**: Configuration language
- **[Zerolog](https://github.com/rs/zerolog)**: Structured logging
- **[samber/do](https://github.com/samber/do)**: Dependency injection
- **[client-go](https://github.com/kubernetes/client-go)**: Kubernetes client library

## Architecture

### Key Components

1. **Runner**: Orchestrates port forwarders, manages context and graceful shutdown
2. **Forwarder**: Implements individual pod port forwarding with automatic reconnection
3. **Config**: CUE-based configuration parsing and validation
4. **Logger**: Structured logging with zerolog
5. **Kubernetes Integration**: Handles kubeconfig loading and client initialization

### Port Forward Flow

1. Read configuration from CUE file
2. Load Kubernetes credentials
3. For each forward:
   - Locate the pod
   - Verify pod is running
   - Establish SPDY connection
   - Forward ports
   - Reconnect on failure (1 second retry delay)
4. Listen for interrupt signal (Ctrl+C)
5. Gracefully shutdown all forwarders

## License

[Add your license here]

## Support

For issues, questions, or contributions, please visit the [repository](https://github.com/your-org/fwkeeper).
