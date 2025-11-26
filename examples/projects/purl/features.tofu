# ========================================
# FEATURE RESOURCES - Multi-Protocol Support
# ========================================

# Protocol abstraction layer feature
resource "tofukit_feature" "protocol_abstraction" {
  name        = "protocol-abstraction"
  description = "Protocol-agnostic pinger interface for extensible multi-protocol support"

  files = {
    "internal/pinger/pinger.go" = tofukit_file.pinger_interface
  }

  requirements = [
    # REQUIREMENT 1: Multi-Protocol Support
    {
      name = "Multi-Protocol Support"

      instructions = [
        {
          prompt = <<-EOF
          Support multiple ping protocols through a unified interface.

          USER CAPABILITY:
          - User can ping targets using HTTP, ICMP, or DNS protocols
          - User can explicitly choose protocol via flags (--http, --icmp, --dns)
          - User can let system auto-detect the appropriate protocol (--auto, default)
          - All protocols return consistent output format

          BUSINESS BEHAVIOR:
          - Same command structure regardless of protocol
          - Consistent output format: timestamp, status, duration
          - All protocols operate continuously (1-second intervals)
          - All protocols are resilient to failures
          EOF

          constraints = [
            "Unified output format across all protocols",
            "Consistent behavior (continuous loop, 1s interval, resilient)",
            "Easy to add new protocols without changing existing ones",
            "Use ${module.stack_go_viper_cobra_pterm.kits.framework.cobra.link} for CLI"
          ]
        }
      ]

      verifications = [
        {
          command = "test -f internal/pinger/pinger.go && echo 'exists'"
          expect  = "exists"
        }
      ]
    },

    # REQUIREMENT 2: Extensibility
    {
      name = "Extensibility"

      instructions = [
        {
          prompt = <<-EOF
          New protocols can be added without modifying existing code.

          BUSINESS GOAL:
          - Adding TCP pinger doesn't require changes to HTTP/ICMP/DNS code
          - Each protocol is independent and self-contained
          - Protocol selection happens dynamically at runtime
          - System can be extended with custom protocols

          DESIGN PRINCIPLE:
          Open-Closed Principle - open for extension, closed for modification.
          EOF

          constraints = [
            "New protocols added via new files only",
            "No changes to existing protocol implementations when adding new ones",
            "Protocol registration is declarative",
            "Follow Go interface patterns"
          ]
        }
      ]

      verifications = []
    }
  ]

  kits = [
    module.stack_go_viper_cobra_pterm.kits.language
  ]

  verifications = []
}

# Protocol detection feature
resource "tofukit_feature" "protocol_detection" {
  name        = "protocol-detection"
  description = "Automatic protocol detection with priority-based selection"

  files = {
    "internal/pinger/detector.go" = tofukit_file.protocol_detector
    "cmd/root.go"                 = tofukit_file.cmd_root
  }

  requirements = [
    # REQUIREMENT 1: Automatic Protocol Selection
    {
      name = "Automatic Protocol Selection"

      instructions = [
        {
          prompt = <<-EOF
          System automatically determines the best protocol for a target.

          This feature builds on ${tofukit_feature.protocol_abstraction.link} to provide
          intelligent protocol selection.

          USER WORKFLOW (--auto flag or no flag):
          1. User runs: ${local.project_name} example.com
          2. System analyzes target:
             - If target starts with http:// or https:// → Use HTTP
             - If target has port 80/443 → Use HTTP
             - If target has port 53 → Use DNS
             - Otherwise, probe in priority order
          3. System probes in priority: HTTP → DNS → ICMP
          4. System stops at first successful probe
          5. System uses detected protocol for continuous pinging

          PROBING BEHAVIOR:
          - HTTP: Check if port 80 or 443 responds to HTTP GET
          - DNS: Check if port 53 responds to DNS query
          - ICMP: Check if host responds to ICMP echo
          - Stop at first success, don't over-probe

          PRIORITY RATIONALE:
          - HTTP provides most information (status codes, headers)
          - DNS is critical infrastructure monitoring
          - ICMP is basic connectivity (lowest level, fallback)
          EOF

          constraints = [
            "Stop probing at first successful detection",
            "Timeout each probe attempt (2-3 seconds max)",
            "Priority order: HTTP > DNS > ICMP (not configurable)",
            "Fall back to ICMP if all else fails"
          ]
        }
      ]

      verifications = [
        {
          command = "grep -q 'func DetectProtocol' internal/pinger/detector.go && echo 'found'"
          expect  = "found"
        }
      ]
    },

    # REQUIREMENT 2: Explicit Protocol Selection
    {
      name = "Explicit Protocol Selection"

      instructions = [
        {
          prompt = <<-EOF
          User can explicitly choose protocol via command-line flags.

          USER WORKFLOW:
          - ${local.project_name} --http example.com → Force HTTP protocol
          - ${local.project_name} --icmp 192.168.1.1 → Force ICMP protocol
          - ${local.project_name} --dns example.com → Force DNS protocol
          - ${local.project_name} example.com → Auto-detect (default)

          BEHAVIOR:
          - Explicit flags bypass auto-detection
          - Invalid protocol for target shows clear error message
          - Example: --icmp http://example.com → Error: "Cannot use ICMP with URL scheme"
          EOF

          constraints = [
            "Flags: --http, --icmp, --dns, --auto (default)",
            "Mutually exclusive flags (only one protocol at a time)",
            "Validate protocol choice is appropriate for target",
            "Use ${module.stack_go_viper_cobra_pterm.kits.framework.cobra.link} for flag handling"
          ]
        }
      ]

      verifications = []
    }
  ]

  kits = [
    module.stack_go_viper_cobra_pterm.kits.framework.cobra,
    module.stack_go_viper_cobra_pterm.kits.language
  ]

  verifications = []
}

# HTTP ping feature (refactored to use abstraction)
resource "tofukit_feature" "http_ping" {
  name        = "http-ping"
  description = "HTTP/HTTPS protocol implementation for multi-protocol architecture"

  files = {
    "internal/pinger/http.go" = tofukit_file.pinger_http
  }

  requirements = [
    # REQUIREMENT 1: HTTP Request Monitoring
    {
      name = "HTTP Request Monitoring"

      instructions = [
        {
          prompt = <<-EOF
          Monitor HTTP/HTTPS endpoints by issuing GET requests.

          This feature implements ${tofukit_feature.protocol_abstraction.link} for HTTP/HTTPS protocol.

          USER WORKFLOW:
          1. User runs: ${local.project_name} http://example.com (or ${local.project_name} --http example.com)
          2. System issues HTTP GET request to URL
          3. System displays for each request:
             - ISO 8601 timestamp
             - HTTP status code (200, 404, 500, etc.)
             - Response time in milliseconds
          4. System color-codes output:
             - Green for 2xx responses (success)
             - Red for 4xx/5xx responses or connection failures
          5. System continues every 1 second until interrupted

          BUSINESS BEHAVIOR:
          - Always use GET method (not POST, PUT, etc.)
          - Follow redirects up to 3 times
          - 10-second timeout per request
          - Continue on failures (don't exit)
          - Log connection errors clearly
          EOF

          constraints = [
            "1 second interval between requests",
            "10 second timeout per request",
            "Continue operation on failures",
            "Use ${module.stack_go_viper_cobra_pterm.patterns.iso8601_timestamp}",
            "Use ${module.stack_go_viper_cobra_pterm.patterns.pterm_output}"
          ]
        }
      ]

      verifications = [
        {
          command = "test -f internal/pinger/http.go && echo 'exists'"
          expect  = "exists"
        }
      ]
    },

    # REQUIREMENT 2: Protocol Detection
    {
      name = "Protocol Detection"

      instructions = [
        {
          prompt = <<-EOF
          HTTP pinger participates in ${tofukit_feature.protocol_detection.link}.

          DETECTION CRITERIA:
          - Target starts with http:// or https:// → Confidence: 100%
          - Target includes port 80 or 443 → Confidence: 90%
          - Port 80/443 is open and responds to HTTP → Confidence: 100%
          - Otherwise → Confidence: 0% (not HTTP)

          DETECTION BEHAVIOR:
          - Try port 80 before port 443 in probing
          - Stop at first successful HTTP response
          - Return confidence score and port used
          EOF

          constraints = [
            "HTTP has highest priority in auto-detection",
            "Probe ports 80, 443 in that order",
            "Quick probe (2-3 second timeout)",
            "Return confidence score 0-100"
          ]
        }
      ]

      verifications = []
    }
  ]

  kits = [
    module.stack_go_viper_cobra_pterm.kits.language
  ]

  verifications = []
}

# ICMP ping feature
resource "tofukit_feature" "icmp_ping" {
  name        = "icmp-ping"
  description = "ICMP echo request/reply for network-level connectivity testing"

  files = {
    "internal/pinger/icmp.go" = tofukit_file.pinger_icmp
  }

  requirements = [
    # REQUIREMENT 1: Network Connectivity Testing
    {
      name = "Network Connectivity Testing"

      instructions = [
        {
          prompt = <<-EOF
          Test network-level connectivity using ICMP echo requests.

          This feature implements ${tofukit_feature.protocol_abstraction.link} for ICMP protocol.

          USER WORKFLOW:
          1. User runs: ${local.project_name} --icmp example.com (or ${local.project_name} 192.168.1.1 with auto-detect)
          2. System resolves hostname to IP address (if needed)
          3. System sends ICMP echo request packet
          4. System displays for each ping:
             - ISO 8601 timestamp
             - Target IP address
             - Round-trip time in milliseconds
             - Success (1) or timeout (0) as status code
          5. System continues every 1 second until interrupted

          BUSINESS BEHAVIOR:
          - Basic connectivity test (reachable or not)
          - Network-level operation (no application layer)
          - Works for any IP address or resolvable hostname
          - Minimal information but universal applicability

          PRIVILEGE NOTICE:
          If ICMP fails due to insufficient privileges, display clear message:
          "ICMP requires elevated privileges. Try: sudo ${local.project_name} --icmp <target>"
          EOF

          constraints = [
            "1 second interval between pings",
            "5 second timeout per ping",
            "StatusCode: 1 for reply received, 0 for timeout",
            "Continue operation on failures",
            "Use ${module.stack_go_viper_cobra_pterm.patterns.iso8601_timestamp}",
            "Use ${module.stack_go_viper_cobra_pterm.patterns.pterm_output}",
            "Use ${module.stack_go_viper_cobra_pterm.patterns.error_handling} for privilege errors"
          ]
        }
      ]

      verifications = [
        {
          command = "test -f internal/pinger/icmp.go && echo 'exists'"
          expect  = "exists"
        }
      ]
    },

    # REQUIREMENT 2: Fallback Protocol
    {
      name = "Fallback Protocol"

      instructions = [
        {
          prompt = <<-EOF
          ICMP serves as lowest-priority fallback in ${tofukit_feature.protocol_detection.link}.

          BUSINESS RATIONALE:
          - ICMP works for any IP address
          - Most basic connectivity test
          - Last resort when HTTP and DNS don't apply
          - Universal but minimal information

          DETECTION BEHAVIOR:
          - ICMP has lowest priority in auto-detection
          - Used when target is plain IP address with no port hints
          - Used when HTTP and DNS probes fail
          - Always available as fallback
          EOF

          constraints = [
            "Lowest detection priority (after HTTP, DNS)",
            "Accepts any IP or resolvable hostname",
            "Confidence score lower than HTTP/DNS",
            "Never rejects a target (universal fallback)"
          ]
        }
      ]

      verifications = []
    }
  ]

  kits = [
    module.stack_go_viper_cobra_pterm.kits.language
  ]

  verifications = []
}

# DNS ping feature
resource "tofukit_feature" "dns_ping" {
  name        = "dns-ping"
  description = "DNS resolution monitoring and performance testing"

  files = {
    "internal/pinger/dns.go" = tofukit_file.pinger_dns
  }

  requirements = [
    # REQUIREMENT 1: DNS Resolution Monitoring
    {
      name = "DNS Resolution Monitoring"

      instructions = [
        {
          prompt = <<-EOF
          Monitor DNS resolution performance and availability.

          This feature implements ${tofukit_feature.protocol_abstraction.link} for DNS protocol.

          USER WORKFLOW:
          1. User runs: ${local.project_name} --dns example.com (or ${local.project_name} example.com:53 with auto-detect)
          2. System performs DNS lookup for hostname
          3. System displays for each query:
             - ISO 8601 timestamp
             - Target hostname
             - DNS server used (system default or specified)
             - Resolution time in milliseconds
             - Number of IP addresses returned (status code)
          4. System continues every 1 second until interrupted

          BUSINESS BEHAVIOR:
          - Measures DNS performance (resolution time)
          - Validates DNS availability (can resolve or not)
          - Tracks DNS health over time
          - Identifies DNS issues before they affect applications

          RESOLUTION OUTCOMES:
          - Success: At least one IP address returned (green output)
          - NXDOMAIN: Domain doesn't exist (red output, but expected failure)
          - Timeout: DNS server unreachable (red output, error)
          - Server failure: DNS server error (red output, error)
          EOF

          constraints = [
            "1 second interval between queries",
            "5 second timeout per query",
            "StatusCode = number of IP addresses returned",
            "NXDOMAIN is expected response, not application error",
            "Continue operation on failures",
            "Use ${module.stack_go_viper_cobra_pterm.patterns.iso8601_timestamp}",
            "Use ${module.stack_go_viper_cobra_pterm.patterns.pterm_output}"
          ]
        }
      ]

      verifications = [
        {
          command = "test -f internal/pinger/dns.go && echo 'exists'"
          expect  = "exists"
        }
      ]
    },

    # REQUIREMENT 2: Custom DNS Server
    {
      name = "Custom DNS Server"

      instructions = [
        {
          prompt = <<-EOF
          Allow querying specific DNS servers for diagnostics.

          USER WORKFLOW:
          - ${local.project_name} --dns example.com → Query system DNS resolver
          - ${local.project_name} --dns example.com@8.8.8.8 → Query Google's DNS
          - ${local.project_name} --dns example.com@1.1.1.1 → Query Cloudflare's DNS

          BUSINESS USE CASES:
          - Compare resolution times across DNS providers
          - Diagnose DNS-specific issues
          - Test DNS propagation after changes
          - Monitor specific DNS server health
          - Verify authoritative vs recursive servers

          FORMAT:
          - "hostname" → Use system resolver
          - "hostname@server" → Use specified DNS server
          - Server can be IP or hostname
          EOF

          constraints = [
            "Parse target for '@' separator",
            "Validate DNS server address",
            "Fall back to system resolver if no server specified",
            "Display which DNS server was used in output"
          ]
        }
      ]

      verifications = []
    },

    # REQUIREMENT 3: Infrastructure Monitoring
    {
      name = "Infrastructure Monitoring"

      instructions = [
        {
          prompt = <<-EOF
          DNS participates in ${tofukit_feature.protocol_detection.link} with medium priority.

          DNS is critical infrastructure that affects all network operations.

          BUSINESS IMPORTANCE:
          - DNS failures masquerade as application problems
          - Slow DNS affects overall application performance
          - DNS issues are often overlooked in monitoring
          - Early DNS problem detection prevents cascading failures

          MONITORING GOALS:
          - Track DNS resolution time trends
          - Detect DNS server failures quickly
          - Identify slow DNS responses
          - Validate DNS changes have propagated
          - Monitor DNS consistency across servers
          EOF

          constraints = [
            "Medium priority in auto-detection (after HTTP, before ICMP)",
            "Detects port 53 or '@server' format",
            "Confidence score between HTTP and ICMP"
          ]
        }
      ]

      verifications = []
    }
  ]

  kits = [
    module.stack_go_viper_cobra_pterm.kits.language
  ]

  verifications = []
}
