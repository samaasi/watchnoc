package pii

import (
	"context"
	"log/slog"
	"reflect"

	"github.com/samaasi/watchnoc/internal/platform/tags"
)

const redactedValue = "[REDACTED]"

// RedactingHandler is a slog.Handler that wraps any underlying handler and
// nullifies PII-tagged fields in structured log attributes before forwarding.
//
// Usage:
//
//	base := slog.NewJSONHandler(os.Stdout, nil)
//	logger := slog.New(pii.NewRedactingHandler(base))
type RedactingHandler struct {
	inner slog.Handler
}

func NewRedactingHandler(inner slog.Handler) *RedactingHandler {
	return &RedactingHandler{inner: inner}
}

func (h *RedactingHandler) Enabled(ctx context.Context, level slog.Level) bool {
	return h.inner.Enabled(ctx, level)
}

func (h *RedactingHandler) Handle(ctx context.Context, r slog.Record) error {
	// Walk the record's attributes and redact PII in any struct values
	cleaned := r.Clone()
	cleaned.Attrs(func(a slog.Attr) bool {
		// Not replacing in place — we build a new record below
		return true
	})

	newRecord := slog.NewRecord(r.Time, r.Level, r.Message, r.PC)
	r.Attrs(func(a slog.Attr) bool {
		newRecord.AddAttrs(redactAttr(a))
		return true
	})

	return h.inner.Handle(ctx, newRecord)
}

func (h *RedactingHandler) WithAttrs(attrs []slog.Attr) slog.Handler {
	redacted := make([]slog.Attr, len(attrs))
	for i, a := range attrs {
		redacted[i] = redactAttr(a)
	}
	return &RedactingHandler{inner: h.inner.WithAttrs(redacted)}
}

func (h *RedactingHandler) WithGroup(name string) slog.Handler {
	return &RedactingHandler{inner: h.inner.WithGroup(name)}
}

// Redactable can be implemented by models to bypass reflection-based PII
// redaction during structured logging, significantly reducing allocations.
// Code generators can emit this method for any struct containing pii:"true".
type Redactable interface {
	RedactPII() interface{}
}

// redactAttr recursively replaces PII-tagged fields with [REDACTED].
func redactAttr(a slog.Attr) slog.Attr {
	switch a.Value.Kind() {
	case slog.KindAny:
		v := a.Value.Any()

		// Fast path: bypass reflection for models that implement Redactable
		if r, ok := v.(Redactable); ok {
			return slog.Attr{Key: a.Key, Value: slog.AnyValue(r.RedactPII())}
		}

		rv := reflect.ValueOf(v)
		if rv.Kind() == reflect.Ptr {
			rv = rv.Elem()
		}
		if rv.Kind() == reflect.Struct {
			cleaned := redactStruct(rv)
			return slog.Attr{Key: a.Key, Value: slog.AnyValue(cleaned.Interface())}
		}
	case slog.KindGroup:
		attrs := a.Value.Group()
		redacted := make([]slog.Attr, len(attrs))
		for i, ga := range attrs {
			redacted[i] = redactAttr(ga)
		}
		return slog.Group(a.Key, attrsToAny(redacted)...)
	}
	return a
}

// redactStruct returns a shallow copy of the struct with PII fields zeroed.
func redactStruct(v reflect.Value) reflect.Value {
	t := v.Type()
	piiFields := tags.FieldsWithTag(t, "pii")
	if len(piiFields) == 0 {
		return v
	}

	// Create a settable copy
	copy := reflect.New(t).Elem()
	copy.Set(v)

	for _, f := range piiFields {
		if f.TagValue != "true" {
			continue
		}
		fv := copy.FieldByName(f.FieldName)
		if !fv.IsValid() || !fv.CanSet() {
			continue
		}
		// Set strings to [REDACTED], pointers to nil, others to zero
		switch fv.Kind() {
		case reflect.String:
			fv.SetString(redactedValue)
		case reflect.Ptr:
			fv.Set(reflect.Zero(fv.Type()))
		default:
			fv.Set(reflect.Zero(fv.Type()))
		}
	}
	return copy
}

func attrsToAny(attrs []slog.Attr) []any {
	out := make([]any, len(attrs))
	for i, a := range attrs {
		out[i] = a
	}
	return out
}
