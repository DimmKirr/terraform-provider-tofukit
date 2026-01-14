package resources

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
	"sync"
	"time"
)

// DebugLog provides file-based debug logging for execution tracing
// Writes to .debug/<timestamp>-tofu-apply.log
type DebugLog struct {
	file      *os.File
	mu        sync.Mutex
	startTime time.Time
	enabled   bool
}

// NewDebugLog creates a new debug logger that writes to the specified output directory
// Returns a disabled logger if debug is disabled or if the log file cannot be created
func NewDebugLog(outputPath string, enabled bool) *DebugLog {
	if !enabled {
		return &DebugLog{enabled: false}
	}

	debugDir := filepath.Join(outputPath, ".debug")
	if err := os.MkdirAll(debugDir, 0755); err != nil {
		log.Printf("[WARN] Failed to create debug directory %s: %v", debugDir, err)
		return &DebugLog{enabled: false}
	}

	timestamp := time.Now().Format("20060102-150405")
	logPath := filepath.Join(debugDir, fmt.Sprintf("%s-tofu-apply.log", timestamp))

	file, err := os.OpenFile(logPath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
	if err != nil {
		log.Printf("[WARN] Failed to create debug log file %s: %v", logPath, err)
		return &DebugLog{enabled: false}
	}

	dl := &DebugLog{
		file:      file,
		startTime: time.Now(),
		enabled:   true,
	}

	// Write header
	dl.write("=== TOFUKIT PROVIDER EXECUTION LOG ===")
	dl.write("Started: %s", dl.startTime.Format(time.RFC3339))
	dl.write("Log file: %s", logPath)
	dl.write("========================================")
	dl.write("")

	return dl
}

// Log writes a timestamped log entry
func (dl *DebugLog) Log(format string, args ...interface{}) {
	if dl == nil || !dl.enabled || dl.file == nil {
		return
	}
	dl.write(format, args...)
}

// Section writes a section header
func (dl *DebugLog) Section(title string) {
	if dl == nil || !dl.enabled || dl.file == nil {
		return
	}
	dl.write("")
	dl.write("--- %s ---", title)
}

// Error writes an error entry with ERROR prefix
func (dl *DebugLog) Error(format string, args ...interface{}) {
	if dl == nil || !dl.enabled || dl.file == nil {
		return
	}
	dl.write("[ERROR] "+format, args...)
}

// Close closes the log file and writes summary
func (dl *DebugLog) Close() {
	if dl == nil || !dl.enabled || dl.file == nil {
		return
	}

	duration := time.Since(dl.startTime)
	dl.write("")
	dl.write("========================================")
	dl.write("Execution completed")
	dl.write("Duration: %v", duration)
	dl.write("Ended: %s", time.Now().Format(time.RFC3339))
	dl.write("========================================")

	dl.file.Close()
}

// write is the internal write method
func (dl *DebugLog) write(format string, args ...interface{}) {
	dl.mu.Lock()
	defer dl.mu.Unlock()

	elapsed := time.Since(dl.startTime)
	timestamp := fmt.Sprintf("[+%9.3fs]", elapsed.Seconds())

	msg := fmt.Sprintf(format, args...)
	line := fmt.Sprintf("%s %s\n", timestamp, msg)

	dl.file.WriteString(line)
}
