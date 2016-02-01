package logrus

import (
	"bytes"
	"fmt"
	"io"
	"runtime"
	"sort"
	"strings"
	"time"
)

const (
	nocolor = 0
	red     = 31
	green   = 32
	yellow  = 33
	blue    = 34
	gray    = 37
)

var (
	baseTimestamp time.Time
)

func init() {
	baseTimestamp = time.Now()
}

type TextFormatter struct {
	// Set to true to bypass checking for a TTY before outputting colors.
	ForceColors bool

	// Force disabling colors.
	DisableColors bool

	// Disable timestamp logging. useful when output is redirected to logging
	// system that already adds timestamps.
	DisableTimestamp bool

	// Enable logging the full timestamp when a TTY is attached instead of just
	// the time passed since beginning of execution.
	FullTimestamp bool

	// TimestampFormat to use for display when a full timestamp is printed
	TimestampFormat string

	// The fields are sorted by default for a consistent output. For applications
	// that log extremely frequently and don't use the JSON formatter this may not
	// be desired.
	DisableSorting bool

	// QuoteEmptyFields will wrap empty fields in quotes if true
	QuoteEmptyFields bool

	// QuoteCharacter can be set to the override the default quoting character "
	// with something else. For example: ', or `.
	QuoteCharacter string
}

// Our internal representation
type textFormatter struct {
	settings  TextFormatter
	isColored bool
}

func (factory *TextFormatter) Build(out io.Writer, minimumLevel Level) (Formatter, error) {
	if !factory.DisableTimestamp && factory.TimestampFormat == "" {
		factory.TimestampFormat = DefaultTimestampFormat
	}
	// TODO more TimestampFormat validation

	if factory.QuoteCharacter == "" {
		factory.QuoteCharacter = `"`
	}

	isColorTerminal := IsTerminal(out) && (runtime.GOOS != "windows")
	return &textFormatter{
		settings: *factory,
		isColored: (factory.ForceColors || isColorTerminal) &&
			!factory.DisableColors,
	}, nil
}

func (f *textFormatter) Format(entry *Entry) ([]byte, error) {
	prefixFieldClashes(entry.Data)
	keys := make([]string, 0, len(entry.Data))
	for k := range entry.Data {
		keys = append(keys, k)
	}

	if !f.settings.DisableSorting {
		sort.Strings(keys)
	}

	b := entry.Buffer
	if b == nil {
		b = bytes.NewBuffer(make([]byte, 0, 200))
	}

	if f.isColored {
		f.printColored(b, entry, keys)
	} else {
		if !f.settings.DisableTimestamp {
			f.appendKeyValue(b, "time", entry.Time.Format(f.settings.TimestampFormat))
		}
		f.appendKeyValue(b, "level", entry.Level.String())
		if entry.Message != "" {
			f.appendKeyValue(b, "msg", entry.Message)
		}
		for _, key := range keys {
			f.appendKeyValue(b, key, entry.Data[key])
		}
	}

	b.WriteByte('\n')
	return b.Bytes(), nil
}

func (f *textFormatter) printColored(b *bytes.Buffer, entry *Entry, keys []string) {
	var levelColor int
	switch entry.Level {
	case DebugLevel:
		levelColor = gray
	case WarnLevel:
		levelColor = yellow
	case ErrorLevel, FatalLevel, PanicLevel:
		levelColor = red
	default:
		levelColor = blue
	}

	levelText := strings.ToUpper(entry.Level.String())[0:4]

	if f.settings.DisableTimestamp {
		fmt.Fprintf(b, "\x1b[%dm%s\x1b[0m %-44s ", levelColor, levelText, entry.Message)
	} else if !f.settings.FullTimestamp {
		fmt.Fprintf(b, "\x1b[%dm%s\x1b[0m[%04d] %-44s ", levelColor, levelText, int(entry.Time.Sub(baseTimestamp)/time.Second), entry.Message)
	} else {
		fmt.Fprintf(b, "\x1b[%dm%s\x1b[0m[%s] %-44s ", levelColor, levelText, entry.Time.Format(f.settings.TimestampFormat), entry.Message)
	}
	for _, k := range keys {
		v := entry.Data[k]
		fmt.Fprintf(b, " \x1b[%dm%s\x1b[0m=", levelColor, k)
		f.appendValue(b, v)
	}
}

func (f *textFormatter) needsQuoting(text string) bool {
	if f.settings.QuoteEmptyFields && len(text) == 0 {
		return true
	}
	for _, ch := range text {
		if !((ch >= 'a' && ch <= 'z') ||
			(ch >= 'A' && ch <= 'Z') ||
			(ch >= '0' && ch <= '9') ||
			ch == '-' || ch == '.') {
			return true
		}
	}
	return false
}

func (f *textFormatter) appendKeyValue(b *bytes.Buffer, key string, value interface{}) {
	b.WriteString(key)
	b.WriteByte('=')
	f.appendValue(b, value)
	b.WriteByte(' ')
}

func (f *textFormatter) appendValue(b *bytes.Buffer, value interface{}) {
	switch value := value.(type) {
	case string:
		if !f.needsQuoting(value) {
			b.WriteString(value)
		} else {
			fmt.Fprintf(b, "%s%v%s", f.settings.QuoteCharacter, value, f.settings.QuoteCharacter)
		}
	case error:
		errmsg := value.Error()
		if !f.needsQuoting(errmsg) {
			b.WriteString(errmsg)
		} else {
			fmt.Fprintf(b, "%s%v%s", f.settings.QuoteCharacter, errmsg, f.settings.QuoteCharacter)
		}
	default:
		fmt.Fprint(b, value)
	}
}
