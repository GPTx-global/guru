# GURU Oracle Daemon

The GURU Oracle Daemon is a distributed oracle system designed to securely and reliably collect external data and submit it to the GURU blockchain network. It provides real-time event processing and high-performance data collection through a CPU-based worker pool architecture.

## System Architecture

### Core Components

The Oracle Daemon consists of four main components that work together to provide a complete oracle service:

```
┌─────────────────────────────────────────────────────────────────────────────┐
│                              Oracle Daemon                                  │
├─────────────────────────────────────────────────────────────────────────────┤
│                                                                             │
│      ┌─────────────────┐    ┌─────────────────┐    ┌─────────────────┐      │
│      │     Monitor     │    │   Scheduler     │    │   Submitter     │      │
│      │                 │    │                 │    │                 │      │
│      │ • Event Sub     │───▶│ • Job Queue     │───▶│ • Tx Building   │      │
│      │ • Real-time     │    │ • Worker Pool   │    │ • Signing       │      │
│      │   Monitoring    │    │ • CPU Scaling   │    │ • Broadcasting  │      │
│      │ • Account Filter│    │ • Result Queue  │    │ • Sequence Mgmt │      │
│      └─────────────────┘    └─────────────────┘    └─────────────────┘      │
│                                                                             │
│   ┌─────────────────────────────────────────────────────────────────────┐   │
│   │                          Executor                                   │   │
│   │                                                                     │   │
│   │ • HTTP Client Pool    • JSON Parsing       • Data Extraction        │   │
│   │ • Retry Mechanism     • Path Navigation    • Error Handling         │   │
│   │ • Connection Reuse    • Array Support      • Type Conversion        │   │
│   └─────────────────────────────────────────────────────────────────────┘   │
└─────────────────────────────────────────────────────────────────────────────┘
                                    │
                                    ▼
┌─────────────────────────────────────────────────────────────────────────────┐
│                         GURU Blockchain Network                             │
├─────────────────────────────────────────────────────────────────────────────┤
│                                                                             │
│     ┌─────────────────┐    ┌─────────────────┐    ┌─────────────────┐       │
│     │  Oracle Module  │    │  Event System   │    │  State Machine  │       │
│     │                 │    │                 │    │                 │       │
│     │ • Request Mgmt  │    │ • Event Emit    │    │ • State Update  │       │
│     │ • Data Verify   │    │ • Subscription  │    │ • Consensus     │       │
│     │ • Result Store  │    │ • Notification  │    │ • Finality      │       │
│     └─────────────────┘    └─────────────────┘    └─────────────────┘       │
└─────────────────────────────────────────────────────────────────────────────┘
```

### Data Flow Pipeline

```
External API
     │
     ▼
┌─────────────────┐    ┌─────────────────┐    ┌─────────────────┐
│  HTTP Request   │───▶│  JSON Parsing   │───▶│ Data Extraction │
│                 │    │                 │    │                 │
│ • GET Request   │    │ • Object Parse  │    │ • Path-based    │
│ • Headers       │    │ • Array Handle  │    │ • Dot Notation  │
│ • Retry Logic   │    │ • Error Check   │    │ • Type Convert  │
└─────────────────┘    └─────────────────┘    └─────────────────┘
                                                       │
                                                       ▼
┌─────────────────┐    ┌─────────────────┐    ┌─────────────────┐
│ Blockchain Sub  │◀───│ Transaction     │◀───│  Result Queue   │
│                 │    │                 │    │                 │
│ • Broadcasting  │    │ • Msg Creation  │    │ • Job Results   │
│ • Response      │    │ • Signing       │    │ • Queue Mgmt    │
│ • Sequence Sync │    │ • Gas Calc      │    │ • Order Ensure  │
└─────────────────┘    └─────────────────┘    └─────────────────┘
```

## Event Processing System

### Event Types and Processing

The Oracle Daemon listens to three types of blockchain events:

#### 1. Register Event (Oracle Request Registration)
```go
// Subscription query
registerQuery := "tm.event='Tx' AND message.action='/guru.oracle.v1.MsgRegisterOracleRequestDoc'"

// Processing flow:
1. Detect new oracle request registration on blockchain
2. Extract account list from request document
3. Verify if current daemon instance is assigned
4. Extract endpoint URL and parsing rules
5. Create Job object and submit to worker pool
```

#### 2. Update Event (Oracle Request Update)
```go
// Subscription query
updateQuery := "tm.event='Tx' AND message.action='/guru.oracle.v1.MsgUpdateOracleRequestDoc'"

// Processing flow:
1. Detect oracle request updates
2. Extract changed configuration
3. Find corresponding job in active job store
4. Update job settings (URL, parsing rules, period)
```

#### 3. Complete Event (Data Collection Completion)
```go
// Subscription query
completeQuery := "tm.event='NewBlock' AND complete_oracle_data_set.request_id EXISTS"

// Processing flow:
1. Detect completion events in new blocks
2. Extract request ID and nonce information
3. Find corresponding job in active job store
4. Synchronize nonce and prepare for next collection cycle
```

## Configuration and Setup

### Home Directory Structure

The Oracle Daemon uses a default home directory structure that is automatically created on first run:

#### Default Home Directory
```bash
# Default location (auto-detected based on OS)
~/.oracled/                    # Base home directory

# Platform-specific locations:
# Linux/macOS: /home/username/.oracled
# Windows:     C:\Users\username\.oracled
```

#### Complete Directory Structure
```bash
~/.oracled/
├── config.toml               # Main configuration file (TOML format)
├── keyring-test/             # Test keyring storage (development)
│   ├── keyring-test.db       # SQLite database for test keys
│   └── *.info                # Key metadata files
├── keyring-file/             # File-based keyring storage (production)
│   ├── *.address             # Address files
│   └── *.info                # Encrypted key files
├── keyring-os/               # OS-native keyring storage (secure)
│   └── (OS-managed storage)  # Platform-specific secure storage
└── logs/                     # Log files directory
    ├── oracled.12345.log     # Current daemon log (PID-based naming)
    └── oracled.*.log         # Historical log files
```

#### Custom Home Directory
```bash
# Override default home directory
./oracled --home /custom/oracle/path
```

### Configuration File Structure

The daemon uses TOML format for configuration with the following structure:

#### Complete Configuration Schema
```toml
# ~/.oracled/config.toml

[chain]
# Blockchain network configuration
id = 'guru_3110-1'                     # Chain ID for the GURU network
endpoint = 'http://localhost:26657'    # RPC endpoint for blockchain connection

[key]
# Cryptographic key management configuration
name = 'oracle-node'                  # Key name for transaction signing
keyring_dir = '/home/user/.oracled'   # Directory for keyring storage
keyring_backend = 'test'              # Keyring backend type (test/file/os)

[gas]
# Transaction fee configuration
limit = 30000                         # Gas limit for transactions
prices = '630000000000aguru'          # Gas price with denomination
```
