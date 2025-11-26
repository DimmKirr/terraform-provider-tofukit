# ========================================
# FILE RESOURCES - Multi-Protocol Support
# ========================================
# Internal pinger package for protocol abstraction

# Protocol interface and factory
resource "tofukit_file" "pinger_interface" {
  name        = "internal/pinger/pinger.go"
  description = "Protocol-agnostic pinger interface and factory"

  instructions = [
    # INSTRUCTION 1: Interface Definition
    {
      prompt = <<-EOF
      Define Go interface for protocol-agnostic ping operations.

      ARCHITECTURE:
      - Interface name: Pinger
      - Methods for: executing ping, checking compatibility, probing ports
      - Result structure: PingResult with timestamp, protocol, target, status, duration
      - Factory function: NewPinger(protocol string) creates appropriate implementation

      DESIGN PATTERN:
      Follow Go interface patterns from standard library (io.Reader, http.Handler examples).
      Use context.Context for cancellation throughout.

      REFERENCE:
      - Go Effective Go documentation on interfaces
      - ${module.stack_go_viper_cobra_pterm.patterns.error_handling}
      EOF

      constraints = [
        "Use Go interface pattern (minimal, focused contract)",
        "All methods accept context.Context for cancellation",
        "PingResult is concrete struct, not interface",
        "Factory returns (Pinger, error) - error for unknown protocols"
      ]
    }
  ]

  verifications = [
    {
      command = "grep -q '^package pinger' internal/pinger/pinger.go"
    },
    {
      command = "grep -q 'type Pinger interface' internal/pinger/pinger.go"
    },
    {
      command = "grep -q 'type PingResult struct' internal/pinger/pinger.go"
    },
    {
      command = "grep -q 'func NewPinger' internal/pinger/pinger.go"
    },
    {
      command = "grep -q 'context\\.Context' internal/pinger/pinger.go"
    }
  ]
}

# Protocol detection and auto-selection logic
resource "tofukit_file" "protocol_detector" {
  name        = "internal/pinger/detector.go"
  description = "Protocol detection logic with priority-based selection"

  instructions = [
    # INSTRUCTION 1: Detection Logic
    {
      prompt = <<-EOF
      Implement protocol auto-detection following the priority sequence from business requirements.

      ARCHITECTURE:
      - Function: DetectProtocol(target string, timeout time.Duration) returns (protocol, port, error)
      - Parse target using net/url.Parse() for URL analysis
      - Probe in priority order: HTTP → DNS → ICMP (stop at first success)
      - Use CanHandle and ProbePort methods from each pinger

      REFERENCE PATTERNS:
      - net/url.Parse() for URL parsing (Go standard library)
      - ${module.stack_go_viper_cobra_pterm.patterns.error_handling}
      - Port detection: Use net.Dial or similar for connectivity checks
      EOF

      constraints = [
        "Stop at first successful probe (don't over-probe)",
        "Use timeout for each probe attempt",
        "Priority: HTTP > DNS > ICMP (hard-coded, not configurable)",
        "Return descriptive error only if all probes fail"
      ]
    },

    # INSTRUCTION 2: Target Parsing Helpers
    {
      prompt = <<-EOF
      Create helper functions for target analysis.

      HELPERS:
      - ParseTarget(target string) extracts scheme, host, port
      - GetDefaultPort(protocol string) returns standard port numbers
      - IsPortOpen(host, port, timeout) checks basic connectivity

      REFERENCE:
      Use Go's net and net/url packages following standard library patterns.
      EOF

      constraints = [
        "Use net/url.Parse() for URL parsing",
        "Handle missing scheme gracefully (common case)",
        "Validate port is numeric if present",
        "Follow ${module.stack_go_viper_cobra_pterm.patterns.error_handling}"
      ]
    }
  ]

  verifications = [
    {
      command = "grep -q '^package pinger' internal/pinger/detector.go"
    },
    {
      command = "grep -q 'func DetectProtocol' internal/pinger/detector.go"
    },
    {
      command = "grep -q 'net/url\\|url\\.Parse' internal/pinger/detector.go"
    }
  ]
}

# HTTP protocol implementation
resource "tofukit_file" "pinger_http" {
  name        = "internal/pinger/http.go"
  description = "HTTP/HTTPS pinger implementation"

  instructions = [
    # INSTRUCTION 1: HTTP Pinger Implementation
    {
      prompt = <<-EOF
      Implement HTTP pinger following Go HTTP client best practices.

      ARCHITECTURE:
      - Type: HTTPPinger (implements ${tofukit_file.pinger_interface.link})
      - Use standard library net/http.Client
      - Implement all Pinger interface methods

      REFERENCE PATTERNS:
      - ${module.stack_go_viper_cobra_pterm.patterns.http_client}
      - ${module.stack_go_viper_cobra_pterm.patterns.error_handling}
      - Go net/http documentation for client usage

      INTEGRATION:
      - Works with ${tofukit_file.protocol_detector.link}
      - Returns PingResult struct from pinger.go
      EOF

      constraints = [
        "Use http.Client with 10-second timeout (per business requirements)",
        "Always close response bodies (defer resp.Body.Close())",
        "Use context for cancellation",
        "Follow ${module.stack_go_viper_cobra_pterm.patterns.http_client}"
      ]
    }
  ]

  verifications = [
    {
      command = "grep -q '^package pinger' internal/pinger/http.go"
    },
    {
      command = "grep -q 'type HTTPPinger struct' internal/pinger/http.go"
    },
    {
      command = "grep -q 'http\\.Client' internal/pinger/http.go"
    },
    {
      command = "grep -q 'context\\.Context' internal/pinger/http.go"
    }
  ]
}

# ICMP protocol implementation
resource "tofukit_file" "pinger_icmp" {
  name        = "internal/pinger/icmp.go"
  description = "ICMP echo request/reply pinger implementation"

  instructions = [
    # INSTRUCTION 1: ICMP Pinger Implementation
    {
      prompt = <<-EOF
      Implement ICMP pinger following Go networking patterns.

      ARCHITECTURE:
      - Type: ICMPPinger (implements ${tofukit_file.pinger_interface.link})
      - Use golang.org/x/net/icmp package for ICMP operations
      - Handle privilege requirements for raw sockets

      REFERENCE:
      - golang.org/x/net/icmp package documentation
      - ${module.stack_go_viper_cobra_pterm.patterns.error_handling}
      - ICMP RFC 792 for protocol details

      PRIVILEGE HANDLING:
      - Detect permission errors (EPERM on Linux, similar on other OS)
      - Provide clear error messages about privilege requirements
      - Document: Linux needs CAP_NET_RAW or root, macOS needs root
      EOF

      constraints = [
        "Use golang.org/x/net/icmp package (standard for Go ICMP)",
        "Implement all Pinger interface methods",
        "Handle DNS resolution before ICMP",
        "StatusCode: 1 for success, 0 for timeout (per business requirements)",
        "Follow ${module.stack_go_viper_cobra_pterm.patterns.error_handling} for privilege errors"
      ]
    }
  ]

  verifications = [
    {
      command = "grep -q '^package pinger' internal/pinger/icmp.go"
    },
    {
      command = "grep -q 'type ICMPPinger struct' internal/pinger/icmp.go"
    },
    {
      command = "grep -q 'golang.org/x/net/icmp\\|x/net/icmp' internal/pinger/icmp.go"
    },
    {
      command = "grep -q 'context\\.Context' internal/pinger/icmp.go"
    }
  ]
}

# DNS protocol implementation
resource "tofukit_file" "pinger_dns" {
  name        = "internal/pinger/dns.go"
  description = "DNS resolution pinger implementation"

  instructions = [
    # INSTRUCTION 1: DNS Pinger Implementation
    {
      prompt = <<-EOF
      Implement DNS pinger following Go DNS resolution patterns.

      ARCHITECTURE:
      - Type: DNSPinger (implements ${tofukit_file.pinger_interface.link})
      - Use standard library net.Resolver or net.LookupHost
      - Support custom DNS server via "hostname@server" format parsing

      REFERENCE:
      - Go net package documentation for DNS operations
      - ${module.stack_go_viper_cobra_pterm.patterns.error_handling}
      - net.Resolver for custom DNS server configuration

      CUSTOM DNS SERVER:
      - Parse target for '@' separator (per business requirements)
      - Format: "hostname@dnsserver" queries specific DNS server
      - Default to system resolver if no '@' present
      EOF

      constraints = [
        "Use net package (standard library) for DNS operations",
        "Implement all Pinger interface methods",
        "Support context for cancellation",
        "StatusCode = number of IP addresses returned (per business requirements)",
        "Handle NXDOMAIN as expected response, not error"
      ]
    }
  ]

  verifications = [
    {
      command = "grep -q '^package pinger' internal/pinger/dns.go"
    },
    {
      command = "grep -q 'type DNSPinger struct' internal/pinger/dns.go"
    },
    {
      command = "grep -q 'net\\.Resolver\\|net\\.LookupHost' internal/pinger/dns.go"
    },
    {
      command = "grep -q 'context\\.Context' internal/pinger/dns.go"
    }
  ]
}

# Updated root command to use multi-protocol abstraction
resource "tofukit_file" "cmd_root_multiprotocol" {
  name        = "cmd/root.go"
  description = "Cobra root command with multi-protocol support and auto-detection"

  instructions = [
    # INSTRUCTION 1: Base Structure (Keep from Stack)
    {
      prompt = <<-EOF
      Extend the Cobra root command structure from the stack.

      BASE STRUCTURE:
      - Start with ${module.stack_go_viper_cobra_pterm.files.cobra_root_cmd.link}
      - Keep all standard elements: package cmd, Execute(), init(), standard flags
      - Maintain cobra.OnInitialize setup

      NO CHANGES to standard stack structure.
      EOF

      constraints = [
        "Preserve all stack initialization logic",
        "Keep standard flags: --config, --log-level, --quiet"
      ]
    },

    # INSTRUCTION 2: Protocol Flags
    {
      prompt = <<-EOF
      Add protocol selection flags to root command.

      FLAGS TO ADD:
      - --http (bool) - Force HTTP protocol
      - --icmp (bool) - Force ICMP protocol
      - --dns (bool) - Force DNS protocol
      - --auto (bool, default true) - Auto-detect protocol

      BEHAVIOR:
      - Flags are mutually exclusive (only one can be true)
      - Default is --auto=true
      - Use ${module.stack_go_viper_cobra_pterm.kits.framework.cobra.link} flag patterns
      EOF

      constraints = [
        "Use Cobra's flag binding (cmd.Flags().BoolVar or similar)",
        "Validate mutual exclusivity in PreRunE or RunE",
        "Default: auto-detection enabled"
      ]
    },

    # INSTRUCTION 3: Multi-Protocol Ping Logic
    {
      prompt = <<-EOF
      Integrate multi-protocol pinger abstraction.

      IMPLEMENTATION:
      1. Parse target from args[0]
      2. Determine protocol:
         - If explicit flag (--http, --icmp, --dns): use that protocol
         - If --auto (default): call DetectProtocol from ${tofukit_file.protocol_detector.link}
      3. Create pinger via NewPinger(protocol) from ${tofukit_file.pinger_interface.link}
      4. Execute continuous ping loop (1-second intervals)
      5. Format output using ${module.stack_go_viper_cobra_pterm.patterns.pterm_output}

      REFERENCE:
      - ${module.stack_go_viper_cobra_pterm.patterns.error_handling} for errors
      - ${module.stack_go_viper_cobra_pterm.patterns.iso8601_timestamp} for timestamps
      EOF

      constraints = [
        "Use pinger abstraction (no protocol-specific code in cmd/root.go)",
        "1 second interval between pings",
        "Continuous loop until Ctrl+C",
        "Follow stack patterns for output formatting"
      ]
    }
  ]

  verifications = [
    {
      command = "grep -q '^package cmd' cmd/root.go"
    },
    {
      command = "grep -q 'internal/pinger' cmd/root.go"
    },
    {
      command = "grep -q '--http\\|--icmp\\|--dns\\|--auto' cmd/root.go"
    },
    {
      command = "grep -q 'func Execute()' cmd/root.go"
    }
  ]
}
