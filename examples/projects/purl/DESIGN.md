# Purl Design Principles

## Multi-Protocol Architecture Pattern

### Overview

Purl implements a **Priority-Based Protocol Detection with Extensible Implementations** pattern. This design allows the tool to support multiple protocols for pinging targets while maintaining clean separation of concerns and easy extensibility.

### Core Concept

When a tool needs to support multiple protocols/methods to interact with a target, and the appropriate protocol isn't always explicit:

1. **Define a common interface** that all protocol implementations satisfy
2. **Implement protocol-specific behavior** behind that interface
3. **Create detection logic** that determines which protocol to use based on:
   - Explicit selection (user flags: `--http`, `--icmp`, `--dns`)
   - Target analysis (URL scheme, port numbers)
   - Probing (actual connection attempts)
4. **Apply priority ordering** - when multiple protocols could work, use a defined priority sequence
5. **Stop at first success** - don't over-probe; use the first protocol that validates successfully

### Architecture Layers

**Three Layers:**

1. **Interface Layer** - Protocol-agnostic contract
   - Define: "Ping something and return result"
   - Any protocol implementation must satisfy this interface
   - Location: `internal/pinger/pinger.go`

2. **Implementation Layer** - Protocol-specific implementations
   - HTTP: Web server pinging (`internal/pinger/http.go`)
   - ICMP: Network-level echo requests (`internal/pinger/icmp.go`)
   - DNS: DNS resolution monitoring (`internal/pinger/dns.go`)
   - Each implementation is independent and self-contained

3. **Detection Layer** - Logic to select the right implementation
   - Location: `internal/pinger/detector.go`
   - Orchestrates protocol selection and validation

### Protocol Detection Flow

```
User runs: purl <target>

┌─────────────────────────────────────┐
│ Explicit flag provided?             │
│ (--http, --icmp, --dns)            │
└──────────┬──────────────────────────┘
           │
    ┌──────┴──────┐
    │ YES         │ NO (auto-detection)
    │             │
    ▼             ▼
┌─────────┐   ┌──────────────────────────┐
│ Use     │   │ Parse target for hints:   │
│ that    │   │ - URL scheme (http://)    │
│ protocol│   │ - Port number (80, 443,   │
│         │   │   53, etc.)               │
└─────────┘   └──────────┬───────────────┘
                         │
                         ▼
              ┌──────────────────────────┐
              │ Probe in priority order:  │
              │ 1. HTTP (ports 80, 443)   │
              │ 2. DNS (port 53)          │
              │ 3. ICMP (no port)         │
              │                           │
              │ For each protocol:        │
              │ - Can it handle target?   │
              │ - Probe port/connection   │
              │ - Success? → USE IT       │
              │ - Failed? → Try next      │
              └──────────┬───────────────┘
                         │
                         ▼
                  ┌─────────────┐
                  │ Execute     │
                  │ continuous  │
                  │ ping loop   │
                  └─────────────┘
```

### Extensibility

**Adding a new protocol (e.g., TCP socket ping):**

1. Create `internal/pinger/tcp.go` implementing the interface
2. Add TCP-specific detection logic (port signatures)
3. Insert into priority sequence in detector
4. No changes needed to existing protocol implementations

**Benefits:**
- Open-Closed Principle: Open for extension, closed for modification
- Single Responsibility: Each protocol implementation handles only its protocol
- Interface Segregation: Minimal interface contract

### Generalization

This pattern applies to any tool where:
- Multiple methods exist to accomplish the same goal
- The "best" method depends on target characteristics
- User might not know which method to use upfront
- Adding new methods should be straightforward

**Real-world examples of this pattern:**
- HTTP client negotiation (HTTP/1.1 vs HTTP/2 vs HTTP/3)
- Database connector selection (MySQL vs PostgreSQL vs SQLite)
- File transfer protocol selection (FTP vs SFTP vs SCP)
- Version control system adaptation (git vs svn vs hg)

### Implementation Guidance for LLM

**Business Layer (Features):**
- Protocol Abstraction Feature: "Support multiple ping protocols through unified interface"
- HTTP Ping Feature: "HTTP/HTTPS endpoint monitoring"
- ICMP Ping Feature: "Network-level echo request/reply"
- DNS Ping Feature: "DNS resolution and response time monitoring"

**Technical Layer (Files):**
- `internal/pinger/pinger.go`: Interface definition and factory function
- `internal/pinger/detector.go`: Protocol detection and priority logic
- `internal/pinger/http.go`: HTTP implementation (move from cmd/root.go)
- `internal/pinger/icmp.go`: ICMP implementation (requires raw sockets)
- `internal/pinger/dns.go`: DNS implementation (use net.LookupHost or dns package)

**Constraints:**
- Use Go interface pattern for protocol abstraction
- Each protocol implementation must be independent (no cross-dependencies)
- Protocol detection must be deterministic (same input → same protocol)
- Adding new protocols must not require changing existing implementations
- Probe in priority order and stop at first success
- Handle errors gracefully (failed probe ≠ application crash)

### Priority Sequence Rationale

**Why HTTP → DNS → ICMP?**

1. **HTTP (Highest Priority)**
   - Most common use case for monitoring web services
   - Richest information (status codes, headers, body)
   - Application-layer visibility

2. **DNS (Medium Priority)**
   - Critical infrastructure component
   - Measures resolution performance
   - Works when HTTP isn't applicable

3. **ICMP (Lowest Priority / Fallback)**
   - Most basic connectivity test
   - Works for any IP address
   - Minimal privileges required (though some systems restrict)
   - Last resort when higher-level protocols unavailable

This ordering optimizes for the most common monitoring scenarios while providing fallback options.
