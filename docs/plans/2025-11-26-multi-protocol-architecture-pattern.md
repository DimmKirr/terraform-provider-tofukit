# Multi-Protocol Architecture Pattern for TofuKit Projects

**Date**: 2025-11-26
**Status**: Design Complete, Implementation Pending
**Example Project**: `examples/projects/purl`

## Overview

This document describes the **Priority-Based Protocol Detection with Extensible Implementations** pattern implemented in the purl project. This pattern enables tools to support multiple protocols/methods for interacting with targets while maintaining clean separation of concerns and extensibility.

## Business Problem

When building network utilities, you often need to support multiple protocols (HTTP, ICMP, DNS, TCP) for pinging or monitoring targets. The challenges are:

1. **Protocol Selection**: User may not know which protocol is appropriate for a target
2. **Extensibility**: Adding new protocols should not require modifying existing code
3. **Consistency**: All protocols should provide consistent output and behavior
4. **Usability**: Should support both explicit selection and auto-detection

## Architecture Pattern

### Three-Layer Design

```
┌─────────────────────────────────────────┐
│     INTERFACE LAYER                     │
│  Protocol-agnostic contract             │
│  (internal/pinger/pinger.go)            │
└─────────────────────────────────────────┘
                  ▲
                  │ implements
                  │
┌─────────────────────────────────────────┐
│     IMPLEMENTATION LAYER                │
│  - HTTP (internal/pinger/http.go)       │
│  - ICMP (internal/pinger/icmp.go)       │
│  - DNS (internal/pinger/dns.go)         │
│  Each implementation is independent     │
└─────────────────────────────────────────┘
                  ▲
                  │ uses
                  │
┌─────────────────────────────────────────┐
│     DETECTION LAYER                     │
│  Protocol selection logic               │
│  (internal/pinger/detector.go)          │
└─────────────────────────────────────────┘
```

### Protocol Detection Flow

```
User Command: purl <target>

1. Explicit Protocol?
   ├─ YES (--http, --icmp, --dns) → Use specified protocol
   └─ NO (--auto or default) → Auto-detect:

      2. Parse Target for Hints
         - URL scheme (http://, https://)
         - Port number (80→HTTP, 443→HTTP, 53→DNS)

      3. Probe in Priority Order (stop at first success)
         ├─ HTTP (ports 80, 443)
         ├─ DNS (port 53)
         └─ ICMP (no port, fallback)

      4. Execute Continuous Ping Loop
```

### Priority Sequence Rationale

**HTTP (Highest) → DNS (Medium) → ICMP (Lowest/Fallback)**

1. **HTTP** - Most common use case, richest information (status codes, headers), application-layer visibility
2. **DNS** - Critical infrastructure, measures resolution performance, works when HTTP isn't applicable
3. **ICMP** - Most basic connectivity, works for any IP, minimal privileges, last resort

## TofuKit Implementation Strategy

### Critical Principle: Business vs Technical Separation

**Features** describe WHAT (user workflows, business behavior):
```hcl
resource "tofukit_feature" "dns_ping" {
  requirements = [{
    name = "DNS Resolution Monitoring"
    instructions = [{
      prompt = <<-EOF
      Monitor DNS resolution performance and availability.

      USER WORKFLOW:
      1. User runs: purl --dns example.com
      2. System performs DNS lookup for hostname
      3. System displays: timestamp, hostname, DNS server, duration, IP count
      4. System continues every 1 second until interrupted

      BUSINESS BEHAVIOR:
      - Measures DNS performance (resolution time)
      - Validates DNS availability (can resolve or not)
      EOF

      constraints = [
        "1 second interval between queries",
        "Use ${module.stack.patterns.iso8601_timestamp}",
        "StatusCode = number of IP addresses returned"
      ]
    }]
  }]
}
```

**Files** describe HOW (using standards, patterns, not prescriptive code):
```hcl
resource "tofukit_file" "pinger_dns" {
  name = "internal/pinger/dns.go"

  instructions = [{
    prompt = <<-EOF
    Implement DNS pinger following Go DNS resolution patterns.

    ARCHITECTURE:
    - Type: DNSPinger (implements ${tofukit_file.pinger_interface.link})
    - Use standard library net.Resolver or net.LookupHost
    - Support custom DNS server via "hostname@server" format

    REFERENCE:
    - Go net package documentation for DNS operations
    - ${module.stack.patterns.error_handling}
    - net.Resolver for custom DNS server configuration
    EOF

    constraints = [
      "Use net package (standard library) for DNS operations",
      "Implement all Pinger interface methods",
      "Handle NXDOMAIN as expected response, not error"
    ]
  }]
}
```

**Key Differences:**
- Features: "User runs X → System does Y → System displays Z"
- Files: "Implement X following Y patterns, reference Z documentation"
- Features: Business requirements and user experience
- Files: Architecture guidance and standard references

### Defer to Standards, Not Prescriptive Code

**❌ Bad (Prescriptive, Step-by-Step Code)**:
```hcl
prompt = <<-EOF
Step 1: Create a struct named DNSPinger with fields resolver *net.Resolver
Step 2: Implement Ping(ctx, target) method that:
  - Calls net.LookupHost(target)
  - Measures time with time.Now() before and after
  - Returns PingResult with timestamp, duration, IP count
Step 3: Handle errors by checking if err != nil...
EOF
```

**✅ Good (Architecture + Standards Reference)**:
```hcl
prompt = <<-EOF
Implement DNS pinger following Go DNS resolution patterns.

ARCHITECTURE:
- Type: DNSPinger (implements Pinger interface)
- Use standard library net.Resolver or net.LookupHost

REFERENCE:
- Go net package documentation for DNS operations
- ${module.stack.patterns.error_handling}

CUSTOM DNS SERVER:
- Parse target for '@' separator (per business requirements)
- Format: "hostname@dnsserver" queries specific DNS server
EOF
```

**Why Defer to Standards?**
1. LLMs have extensive knowledge of Go standard library and best practices
2. Standards evolve; references stay current while prescriptive code becomes outdated
3. Prescriptive code is brittle and doesn't adapt to context
4. Stack patterns provide project-specific conventions without over-specifying

### Resource URI Linking

Use TofuKit's URI system to create explicit relationships:

```hcl
# Define interface
resource "tofukit_file" "pinger_interface" {
  name = "internal/pinger/pinger.go"
  # ...
}

# Reference interface in implementation
resource "tofukit_file" "pinger_http" {
  name = "internal/pinger/http.go"
  instructions = [{
    prompt = <<-EOF
    Type: HTTPPinger (implements ${tofukit_file.pinger_interface.link})
    EOF
  }]
}
```

URIs (like `tofukit://file/pinger_interface`) provide metadata to Claude about dependencies.

## Extensibility Benefits

Adding a new protocol (e.g., TCP socket ping):

1. ✅ Create `internal/pinger/tcp.go` implementing the interface
2. ✅ Add TCP detection logic to detector
3. ✅ Insert into priority sequence
4. ✅ **No changes to existing protocol implementations**

This follows the **Open-Closed Principle**: Open for extension, closed for modification.

## File Structure

```
examples/projects/purl/
├── DESIGN.md                        # Pattern explanation
├── project.tofu                     # Provider config, main project
├── features.tofu                    # Original HTTP-only feature
├── files.tofu                       # Original HTTP-only file
├── features-multiprotocol.tofu      # Multi-protocol features (537 lines)
│   ├── protocol_abstraction         # Interface and factory
│   ├── protocol_detection           # Auto-detection logic
│   ├── http_ping_multiprotocol      # HTTP implementation
│   ├── icmp_ping                    # ICMP implementation
│   └── dns_ping                     # DNS implementation
└── files-multiprotocol.tofu         # Multi-protocol files (376 lines)
    ├── pinger_interface             # Go interface definition
    ├── protocol_detector            # Detection and priority logic
    ├── pinger_http                  # HTTP pinger
    ├── pinger_icmp                  # ICMP pinger (requires golang.org/x/net/icmp)
    ├── pinger_dns                   # DNS pinger
    └── cmd_root_multiprotocol       # Updated root command
```

## Lessons Learned

### 1. Business vs Technical Separation is Critical

**Before Refactoring**: Features and files both had prescriptive technical details, causing duplication and confusion.

**After Refactoring**:
- Features describe user workflows and business behavior
- Files reference standards, patterns, and provide architecture guidance
- Result: Clearer separation, less duplication, more maintainable

### 2. Stack Patterns Reduce Duplication

Extracting common patterns to the stack (http_client, error_handling, iso8601_timestamp, pterm_output) allows all projects to reference them consistently:

```hcl
constraints = [
  "Use ${module.stack.patterns.http_client}",
  "Use ${module.stack.patterns.error_handling}"
]
```

### 3. Trust LLM Knowledge, Provide Architecture

Instead of writing step-by-step code instructions, provide:
- Architecture requirements (interfaces, types, structure)
- References to standard libraries and documentation
- Business constraints from requirements
- Links to stack patterns for project conventions

The LLM has deep knowledge of Go patterns, standard library APIs, and best practices. Architecture guidance is more valuable than code prescriptions.

### 4. Resource URIs Create Explicit Dependencies

Using `${tofukit_file.pinger_interface.link}` makes dependencies explicit and provides Claude with context about related resources.

## Generalization: When to Use This Pattern

This pattern applies when:
- ✅ Multiple methods exist to accomplish the same goal
- ✅ The "best" method depends on target characteristics
- ✅ User might not know which method to use upfront
- ✅ Adding new methods should be straightforward

**Real-world examples**:
- HTTP client negotiation (HTTP/1.1 vs HTTP/2 vs HTTP/3)
- Database connector selection (MySQL vs PostgreSQL vs SQLite)
- File transfer protocol selection (FTP vs SFTP vs SCP)
- Authentication method selection (OAuth vs JWT vs API key)
- Version control system adaptation (git vs svn vs hg)

## Implementation Status

- ✅ Design documented (DESIGN.md in purl project)
- ✅ Features defined (features-multiprotocol.tofu)
- ✅ Files defined (files-multiprotocol.tofu)
- ✅ Business/technical separation validated
- ⏳ Integration into main project.tofu (pending)
- ⏳ E2E testing (pending)
- ⏳ Git commit (pending)

## Next Steps

1. Update `project.tofu` to reference multi-protocol features
2. Run `tofu plan` to validate configuration
3. Run `tofu apply` to generate code
4. Test multi-protocol functionality
5. Run E2E test to verify
6. Commit multi-protocol implementation

## References

- Example implementation: `examples/projects/purl/`
- Stack patterns: `examples/stacks/tofukit-stack-go-viper-cobra-pterm/common-patterns.tofu`
- TofuKit resource URIs: `internal/uri/` in provider codebase
- Go interface patterns: [Effective Go - Interfaces](https://go.dev/doc/effective_go#interfaces)
