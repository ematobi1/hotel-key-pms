package logger

import (
	"regexp"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

// sensitivePatterns are compiled once and used by the redact encoder wrapper.
// They cover credit card numbers, email addresses, phone numbers, and
// passport numbers commonly appearing in hotel guest data.
var sensitivePatterns = []*regexp.Regexp{
	regexp.MustCompile(`\b(?:4[0-9]{12}(?:[0-9]{3})?|5[1-5][0-9]{14}|3[47][0-9]{13}|3(?:0[0-5]|[68][0-9])[0-9]{11}|6(?:011|5[0-9]{2})[0-9]{12})\b`), // credit card
	regexp.MustCompile(`[a-zA-Z0-9._%+\-]+@[a-zA-Z0-9.\-]+\.[a-zA-Z]{2,}`),                                                                              // email
	regexp.MustCompile(`(?:\+?[0-9]{1,3}[\s\-]?)?(?:\([0-9]{1,4}\)[\s\-]?)?[0-9]{3,4}[\s\-]?[0-9]{3,4}[\s\-]?[0-9]{3,4}`),                             // phone
	regexp.MustCompile(`[A-Z]{1,2}[0-9]{6,9}`),                                                                                                           // passport
}

const redacted = "[REDACTED]"

// New returns a production zap logger with a redacting string encoder.
// The redaction runs on every string field value before it is written to the output.
func New() (*zap.Logger, error) {
	cfg := zap.NewProductionConfig()
	cfg.EncoderConfig.EncodeTime = zapcore.ISO8601TimeEncoder

	log, err := cfg.Build(zap.WrapCore(func(core zapcore.Core) zapcore.Core {
		return &redactingCore{Core: core}
	}))
	if err != nil {
		return nil, err
	}
	return log, nil
}

type redactingCore struct {
	zapcore.Core
}

func (r *redactingCore) With(fields []zapcore.Field) zapcore.Core {
	return &redactingCore{Core: r.Core.With(redactFields(fields))}
}

func (r *redactingCore) Check(entry zapcore.Entry, ce *zapcore.CheckedEntry) *zapcore.CheckedEntry {
	if r.Enabled(entry.Level) {
		return ce.AddCore(entry, r)
	}
	return ce
}

func (r *redactingCore) Write(entry zapcore.Entry, fields []zapcore.Field) error {
	return r.Core.Write(entry, redactFields(fields))
}

func redactFields(fields []zapcore.Field) []zapcore.Field {
	out := make([]zapcore.Field, len(fields))
	for i, f := range fields {
		if f.Type == zapcore.StringType {
			f.String = redactString(f.String)
		}
		out[i] = f
	}
	return out
}

func redactString(s string) string {
	for _, re := range sensitivePatterns {
		s = re.ReplaceAllString(s, redacted)
	}
	return s
}
