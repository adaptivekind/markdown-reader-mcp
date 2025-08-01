package main

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
)

// ANSI color codes
const (
	colorReset  = "\033[0m"
	colorRed    = "\033[31m"
	colorYellow = "\033[33m"
	colorBlue   = "\033[34m"
	colorGray   = "\033[37m"
	colorCyan   = "\033[36m"
	colorGreen  = "\033[32m"
)

type prettyHandler struct {
	handler slog.Handler
	writer  io.Writer
}

func newPrettyHandler(w io.Writer, opts *slog.HandlerOptions) *prettyHandler {
	return &prettyHandler{
		handler: slog.NewTextHandler(w, opts),
		writer:  w,
	}
}

func (h *prettyHandler) Enabled(ctx context.Context, level slog.Level) bool {
	return h.handler.Enabled(ctx, level)
}

func (h *prettyHandler) WithAttrs(attrs []slog.Attr) slog.Handler {
	return &prettyHandler{
		handler: h.handler.WithAttrs(attrs),
		writer:  h.writer,
	}
}

func (h *prettyHandler) WithGroup(name string) slog.Handler {
	return &prettyHandler{
		handler: h.handler.WithGroup(name),
		writer:  h.writer,
	}
}

func (h *prettyHandler) Handle(ctx context.Context, r slog.Record) error {
	// Get level color
	var levelColor string
	var levelName string
	switch r.Level {
	case slog.LevelDebug:
		levelColor = colorGray
		levelName = "DEBUG"
	case slog.LevelInfo:
		levelColor = colorBlue
		levelName = "INFO "
	case slog.LevelWarn:
		levelColor = colorYellow
		levelName = "WARN "
	case slog.LevelError:
		levelColor = colorRed
		levelName = "ERROR"
	default:
		levelColor = colorReset
		levelName = r.Level.String()
	}

	// Format time
	timeStr := r.Time.Format("15:04:05.000")

	// Start building the log line
	var sb strings.Builder
	sb.WriteString(colorGray)
	sb.WriteString(timeStr)
	sb.WriteString(colorReset)
	sb.WriteString(" ")
	sb.WriteString(levelColor)
	sb.WriteString(levelName)
	sb.WriteString(colorReset)
	sb.WriteString(" ")
	sb.WriteString(r.Message)

	// Add attributes
	r.Attrs(func(a slog.Attr) bool {
		sb.WriteString(" ")
		sb.WriteString(colorCyan)
		sb.WriteString(a.Key)
		sb.WriteString(colorReset)
		sb.WriteString("=")
		sb.WriteString(colorGreen)

		// Handle different value types
		switch v := a.Value.Any().(type) {
		case string:
			sb.WriteString(fmt.Sprintf("%q", v))
		case error:
			sb.WriteString(fmt.Sprintf("%q", v.Error()))
		default:
			sb.WriteString(fmt.Sprintf("%v", v))
		}
		sb.WriteString(colorReset)
		return true
	})

	sb.WriteString("\n")

	_, err := h.writer.Write([]byte(sb.String()))
	return err
}

func configureLogger() {
	logLevel := slog.LevelInfo // Default to info, warnings and errors

	// Determine debug logging setting with command line flags taking precedence
	debugLogging := config.DebugLogging
	if *debugFlag {
		debugLogging = true // Command line --debug overrides config
	} else if *quietFlag {
		debugLogging = false // Command line --quiet overrides config
	}

	if debugLogging {
		logLevel = slog.LevelDebug // Show debug messages when enabled
	}

	// Determine log output destination
	var logOutput *os.File

	if config.LogFile != "" {
		// Expand tilde in log file path
		logPath, err := expandTilde(config.LogFile)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Warning: Could not expand log file path %s: %v\n", config.LogFile, err)
			logOutput = os.Stderr
		} else {
			// Create directory if it doesn't exist
			logDir := filepath.Dir(logPath)
			if err := os.MkdirAll(logDir, 0755); err != nil {
				fmt.Fprintf(os.Stderr, "Warning: Could not create log directory %s: %v\n", logDir, err)
				logOutput = os.Stderr
			} else {
				// Open log file for writing (create or append)
				logOutput, err = os.OpenFile(logPath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
				if err != nil {
					fmt.Fprintf(os.Stderr, "Warning: Could not open log file %s: %v\n", logPath, err)
					logOutput = os.Stderr
				}
			}
		}
	} else {
		logOutput = os.Stderr
	}

	logger = slog.New(newPrettyHandler(logOutput, &slog.HandlerOptions{Level: logLevel}))
}
