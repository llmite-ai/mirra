package logger

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"strings"
	"time"
)

// ANSI color codes
const (
	colorReset      = "\033[0m"
	colorGrey       = "\033[90m"
	colorBlue       = "\033[94m"
	colorGreen      = "\033[92m"
	colorYellow     = "\033[93m"
	colorRed        = "\033[91m"
	colorOrange     = "\033[38;5;208m"
	colorBrightBlue = "\033[96m"
	colorPurple     = "\033[95m"
)

// PrettyHandler is a human-readable slog.Handler implementation
type PrettyHandler struct {
	w     io.Writer
	level slog.Level
	attrs []slog.Attr
}

// NewPrettyHandler creates a new pretty handler
func NewPrettyHandler(w io.Writer, level slog.Level) *PrettyHandler {
	return &PrettyHandler{
		w:     w,
		level: level,
		attrs: []slog.Attr{},
	}
}

// Enabled reports whether the handler handles records at the given level
func (h *PrettyHandler) Enabled(_ context.Context, level slog.Level) bool {
	return level >= h.level
}

// Handle formats and writes the log record
func (h *PrettyHandler) Handle(_ context.Context, r slog.Record) error {
	var b strings.Builder

	// Check if this is a request completion log
	if r.Message == "request completed" {
		h.formatRequestLog(&b, r)
	} else {
		h.formatStandardLog(&b, r)
	}

	b.WriteString("\n")

	_, err := h.w.Write([]byte(b.String()))
	return err
}

// formatRequestLog formats proxy request logs with special styling
func (h *PrettyHandler) formatRequestLog(b *strings.Builder, r slog.Record) {
	// Extract attributes
	attrs := make(map[string]any)
	r.Attrs(func(attr slog.Attr) bool {
		attrs[attr.Key] = attr.Value.Any()
		return true
	})

	// [TIME in grey] ⛁ [ID in blue] [provider with logo] duration [status]

	// Timestamp in grey
	b.WriteString(colorGrey)
	b.WriteString("[")
	b.WriteString(r.Time.Format("15:04:05"))
	b.WriteString("]")
	b.WriteString(colorReset)
	b.WriteString(" ")

	// Taco symbol
	b.WriteString("⛁")
	b.WriteString(" ")

	// ID in bright blue
	if id, ok := attrs["id"].(string); ok {
		b.WriteString(colorBrightBlue)
		b.WriteString(id)
		b.WriteString(colorReset)
		b.WriteString(" ")
	}

	// Provider with colored logo/emoji
	if provider, ok := attrs["provider"].(string); ok {
		switch provider {
		case "claude":
			b.WriteString(colorOrange)
			b.WriteString("❈ claude")
			b.WriteString(colorReset)
		case "openai":
			b.WriteString(colorGrey)
			b.WriteString("● openai")
			b.WriteString(colorReset)
		case "gemini":
			b.WriteString(colorPurple)
			b.WriteString("𝗚 gemini")
			b.WriteString(colorReset)
		default:
			b.WriteString(provider)
		}
		b.WriteString(" ")
	}

	// Duration (human-readable)
	if durationMs, ok := attrs["duration_ms"].(int64); ok {
		b.WriteString(h.formatDuration(durationMs))
		b.WriteString(" ")
	}

	// Status code with color
	var status int
	if s, ok := attrs["status"].(int); ok {
		status = s
	} else if s, ok := attrs["status"].(int64); ok {
		status = int(s)
	}

	if status > 0 {
		statusColor := colorGreen
		if status >= 500 {
			statusColor = colorRed
		} else if status >= 400 {
			statusColor = colorYellow
		} else if status >= 300 {
			statusColor = colorBlue
		}

		b.WriteString(statusColor)
		b.WriteString(fmt.Sprintf("%d", status))
		b.WriteString(colorReset)
		b.WriteString(" ")
	}

	// Path in grey
	if path, ok := attrs["path"].(string); ok {
		b.WriteString(colorGrey)
		b.WriteString(path)
		b.WriteString(colorReset)
	}
}

// formatStandardLog formats standard log messages
func (h *PrettyHandler) formatStandardLog(b *strings.Builder, r slog.Record) {
	// Timestamp in grey
	b.WriteString(colorGrey)
	b.WriteString("[")
	b.WriteString(r.Time.Format("15:04:05"))
	b.WriteString("]")
	b.WriteString(colorReset)
	b.WriteString(" ")

	// Level with color/symbol
	levelStr := h.formatLevel(r.Level)
	b.WriteString(levelStr)
	b.WriteString(" ")

	// Message
	b.WriteString(r.Message)

	// Attributes from handler
	for _, attr := range h.attrs {
		b.WriteString(" ")
		h.appendAttr(b, attr)
	}

	// Attributes from record
	r.Attrs(func(attr slog.Attr) bool {
		b.WriteString(" ")
		h.appendAttr(b, attr)
		return true
	})
}

// formatDuration converts milliseconds to human-readable format
func (h *PrettyHandler) formatDuration(ms int64) string {
	if ms < 1000 {
		return fmt.Sprintf("%dms", ms)
	}
	seconds := float64(ms) / 1000.0
	if seconds < 60 {
		return fmt.Sprintf("%.1fs", seconds)
	}
	minutes := int(seconds / 60)
	remainingSeconds := seconds - float64(minutes*60)
	return fmt.Sprintf("%dm%.0fs", minutes, remainingSeconds)
}

// WithAttrs returns a new handler with the given attributes
func (h *PrettyHandler) WithAttrs(attrs []slog.Attr) slog.Handler {
	newAttrs := make([]slog.Attr, len(h.attrs)+len(attrs))
	copy(newAttrs, h.attrs)
	copy(newAttrs[len(h.attrs):], attrs)

	return &PrettyHandler{
		w:     h.w,
		level: h.level,
		attrs: newAttrs,
	}
}

// WithGroup returns a new handler with the given group
func (h *PrettyHandler) WithGroup(name string) slog.Handler {
	// For simplicity, we'll prefix attrs with the group name
	return &PrettyHandler{
		w:     h.w,
		level: h.level,
		attrs: append(h.attrs, slog.String("group", name)),
	}
}

// formatLevel formats the log level with visual indicators and colors
func (h *PrettyHandler) formatLevel(level slog.Level) string {
	switch level {
	case slog.LevelDebug:
		return colorGrey + "▪ debug" + colorReset
	case slog.LevelInfo:
		return colorBlue + "◆ info" + colorReset
	case slog.LevelWarn:
		return colorYellow + "▲ warn" + colorReset
	case slog.LevelError:
		return colorRed + "● error" + colorReset
	default:
		return fmt.Sprintf("%-5s", level.String())
	}
}

// appendAttr formats and appends an attribute to the builder
func (h *PrettyHandler) appendAttr(b *strings.Builder, attr slog.Attr) {
	key := attr.Key
	val := attr.Value

	// Color the key in grey
	b.WriteString(colorGrey)
	b.WriteString(key)
	b.WriteString("=")
	b.WriteString(colorReset)

	// Handle different value types
	switch val.Kind() {
	case slog.KindString:
		fmt.Fprintf(b, "%q", val.String())
	case slog.KindInt64:
		fmt.Fprintf(b, "%d", val.Int64())
	case slog.KindUint64:
		fmt.Fprintf(b, "%d", val.Uint64())
	case slog.KindFloat64:
		fmt.Fprintf(b, "%.2f", val.Float64())
	case slog.KindBool:
		fmt.Fprintf(b, "%t", val.Bool())
	case slog.KindDuration:
		fmt.Fprintf(b, "%s", val.Duration())
	case slog.KindTime:
		fmt.Fprintf(b, "%s", val.Time().Format(time.RFC3339))
	default:
		fmt.Fprintf(b, "%v", val.Any())
	}
}

// NewLogger creates a new slog.Logger based on format and level
func NewLogger(format, level string, w io.Writer) *slog.Logger {
	logLevel := parseLevel(level)

	var handler slog.Handler
	switch format {
	case "json":
		handler = slog.NewJSONHandler(w, &slog.HandlerOptions{Level: logLevel})
	case "plain":
		handler = slog.NewTextHandler(w, &slog.HandlerOptions{Level: logLevel})
	case "pretty":
		handler = NewPrettyHandler(w, logLevel)
	default:
		handler = NewPrettyHandler(w, logLevel)
	}

	return slog.New(handler)
}

// parseLevel converts a string level to slog.Level
func parseLevel(level string) slog.Level {
	switch strings.ToLower(level) {
	case "debug":
		return slog.LevelDebug
	case "info":
		return slog.LevelInfo
	case "warn", "warning":
		return slog.LevelWarn
	case "error":
		return slog.LevelError
	default:
		return slog.LevelInfo
	}
}
