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
      - Protocol-agnostic interface for ping operations
      - Result structure with timing and status information
      - Factory function for creating protocol-specific implementations

      DESIGN PATTERN:
      Follow Go interface patterns from standard library (io.Reader, http.Handler examples).
      Use context.Context for cancellation throughout.

      REFERENCE:
      - Go Effective Go documentation on interfaces
      - ${module.stack_go_viper_cobra_pterm.features.error_handling_pattern.link}
      EOF

      constraints = []
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
      - Function: DetectProtocol(target, timeout) returns protocol selection
      - Probe in priority order: HTTP → DNS → ICMP (stop at first success)
      - Use pinger implementations from ${tofukit_file.pinger_interface.link}

      REFERENCE:
      - Go net/url package for URL parsing
      - Go net package for connectivity checks
      - ${module.stack_go_viper_cobra_pterm.features.error_handling_pattern.link}
      EOF

      constraints = [
        "Stop at first successful probe (don't over-probe)",
        "Priority: HTTP > DNS > ICMP (hard-coded, not configurable)"
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
      - ${module.stack_go_viper_cobra_pterm.features.http_client_pattern.link}
      - ${module.stack_go_viper_cobra_pterm.features.error_handling_pattern.link}
      - Go net/http documentation for client usage

      INTEGRATION:
      - Works with ${tofukit_file.protocol_detector.link}
      - Returns PingResult struct from pinger.go
      EOF

      constraints = [
        "Timeout configured via ${tofukit_feature.user_config.link}"
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

      REFERENCE:
      - golang.org/x/net/icmp package documentation
      - ${module.stack_go_viper_cobra_pterm.features.error_handling_pattern.link}
      - ICMP RFC 792 for protocol details
      EOF

      constraints = [
        "StatusCode: 1 for success, 0 for timeout (per business requirements)"
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
      - ${module.stack_go_viper_cobra_pterm.features.error_handling_pattern.link}
      - net.Resolver for custom DNS server configuration

      CUSTOM DNS SERVER:
      - Parse target for '@' separator (per business requirements)
      - Format: "hostname@dnsserver" queries specific DNS server
      - Default to system resolver if no '@' present
      EOF

      constraints = [
        "StatusCode = number of IP addresses returned (per business requirements)"
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
resource "tofukit_file" "cmd_root" {
  name        = "cmd/root.go"
  description = "Cobra root command with multi-protocol support and auto-detection"

  instructions = [
    # INSTRUCTION 1: Base Structure (Keep from Stack)
    {
      prompt = <<-EOF
      Extend the Cobra root command structure from the stack.

      BASE STRUCTURE:
      - Extend ${module.stack_go_viper_cobra_pterm.files.cobra_root_cmd.link}
      - Keep all standard elements: package cmd, Execute(), init()
      - Preserve standard flags from ${module.stack_go_viper_cobra_pterm.files.cobra_root_cmd.link}
      - Maintain cobra.OnInitialize setup
      - Import frameworks: ${module.stack_go_viper_cobra_pterm.kits.framework.cobra.link}, ${module.stack_go_viper_cobra_pterm.kits.framework.viper.link}

      NO CHANGES to standard stack structure.
      EOF

      constraints = []
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
        "Flags are mutually exclusive (validate per Cobra best practices)"
      ]
    },

    # INSTRUCTION 3: Multi-Protocol Ping Logic
    {
      prompt = <<-EOF
      Integrate multi-protocol pinger abstraction with protocol selection.

      ARCHITECTURE:
      - Use ${tofukit_file.protocol_detector.link} for protocol selection (when --auto)
      - Use ${tofukit_file.pinger_interface.link} factory for pinger creation
      - Implement continuous ping loop following Go patterns

      OUTPUT FORMATTING:
      - ${module.stack_go_viper_cobra_pterm.features.pterm_output_pattern.link} for colored output
      - ${module.stack_go_viper_cobra_pterm.features.iso8601_timestamp_pattern.link} for timestamps
      - ${module.stack_go_viper_cobra_pterm.features.error_handling_pattern.link} for error messages

      REFERENCE:
      Follow ${module.stack_go_viper_cobra_pterm.kits.framework.cobra.link} patterns for command execution.
      EOF

      constraints = [
        "Use pinger abstraction (no protocol-specific code in cmd/root.go)"
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
