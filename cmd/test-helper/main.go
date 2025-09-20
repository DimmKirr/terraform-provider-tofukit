package main

import (
	"flag"
	"fmt"
	"log"
	"os"

	"github.com/tofukit/opentofu-provider-tofukit/internal/testutil"
)

func main() {
	var (
		testType     = flag.String("type", "quick", "Type of test to run: quick, hello, complex, validate")
		dryRun       = flag.Bool("dry-run", false, "Run in dry-run mode")
		outputDir    = flag.String("output", "./test-output", "Output directory for test results")
		projectName  = flag.String("name", "", "Project name for custom test")
		instructions = flag.String("instructions", "", "Custom instructions for test")
	)
	flag.Parse()

	fmt.Printf("🧪 TofuKit Provider Test Helper\n")
	fmt.Printf("================================\n\n")

	switch *testType {
	case "quick":
		fmt.Printf("Running quick test...\n")
		if err := testutil.RunQuickTest(); err != nil {
			log.Fatalf("Quick test failed: %v", err)
		}

	case "hello":
		fmt.Printf("Running hello world test (dry-run: %t)...\n", *dryRun)
		if err := testutil.RunHelloWorldTest(*dryRun); err != nil {
			log.Fatalf("Hello world test failed: %v", err)
		}

	case "complex":
		fmt.Printf("Running complex project test...\n")
		if err := testutil.RunComplexProjectTest(); err != nil {
			log.Fatalf("Complex test failed: %v", err)
		}

	case "validate":
		fmt.Printf("Running Claude CLI validation...\n")
		if err := testutil.TestClaudeValidation(); err != nil {
			log.Fatalf("Claude validation failed: %v", err)
		}

	case "custom":
		if *projectName == "" || *instructions == "" {
			log.Fatal("Custom test requires both -name and -instructions flags")
		}

		config := testutil.DefaultManualTestConfig()
		config.ProjectName = *projectName
		config.Instructions = *instructions
		config.OutputDir = *outputDir
		config.DryRun = *dryRun

		fmt.Printf("Running custom test: %s\n", *projectName)
		if err := testutil.RunManualTest(config); err != nil {
			log.Fatalf("Custom test failed: %v", err)
		}

	default:
		fmt.Printf("❌ Unknown test type: %s\n", *testType)
		fmt.Printf("\nAvailable test types:\n")
		fmt.Printf("  quick    - Quick hello.txt test (dry-run)\n")
		fmt.Printf("  hello    - Hello world test\n")
		fmt.Printf("  complex  - Complex multi-requirement test\n")
		fmt.Printf("  validate - Validate Claude CLI setup\n")
		fmt.Printf("  custom   - Custom test (requires -name and -instructions)\n")
		fmt.Printf("\nUsage examples:\n")
		fmt.Printf("  go run cmd/test-helper/main.go -type=quick\n")
		fmt.Printf("  go run cmd/test-helper/main.go -type=hello -dry-run=false\n")
		fmt.Printf("  go run cmd/test-helper/main.go -type=validate\n")
		fmt.Printf("  go run cmd/test-helper/main.go -type=custom -name=my-test -instructions=\"Create a Go web server\"\n")
		os.Exit(1)
	}

	fmt.Printf("\n✅ Test completed successfully!\n")
}
