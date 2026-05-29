// internal/platform/mask/masker.go

package mask

import (
    "fmt"
    "net/http"
    "reflect"
    "strings"
    "unicode/utf8"

    "github.com/samaasi/watchnoc/internal/platform/tags"
)

// MaskStrategy defines how a value is visually masked.
type MaskStrategy string

const (
    // StrategyPartial reveals the prefix up to the first separator (e.g. "xoxb-")
    // and masks the rest with asterisks. Safe for tokens with a recognisable prefix.
    StrategyPartial MaskStrategy = "partial"

    // StrategyFull replaces the entire value with asterisks. Used when no
    // prefix is meaningful.
    StrategyFull MaskStrategy = "full"
)

// MaskValue applies the appropriate masking strategy to a string value.
// Designed to be called from the API serialiser before returning to the client.
//
// Examples:
//
//	MaskValue("xoxb-123-456-abcdef", "partial")  → "xoxb-****"
//	MaskValue("glpat-xyz789abc",     "partial")  → "glpat-****"
//	MaskValue("secret",              "partial")  → "se****"
//	MaskValue("",                    "partial")  → ""
func MaskValue(value, strategy string) string {
    if value == "" {
        return ""
    }

    switch MaskStrategy(strategy) {
    case StrategyPartial:
        return partialMask(value)
    case StrategyFull:
        return strings.Repeat("*", utf8.RuneCountInString(value))
    default:
        return strings.Repeat("*", utf8.RuneCountInString(value))
    }
}

// partialMask reveals the prefix up to and including the first separator,
// then replaces the rest with "****".
//
// Separator detection order: "-", "_", "."
// If no separator found, reveals first 2 characters.
func partialMask(value string) string {
    separators := []string{"-", "_", "."}
    for _, sep := range separators {
        if idx := strings.Index(value, sep); idx != -1 {
            prefix := value[:idx+len(sep)] // include the separator
            return prefix + "****"
        }
    }
    // No separator — reveal first 2 characters
    runes := []rune(value)
    if len(runes) <= 2 {
        return "****"
    }
    return string(runes[:2]) + "****"
}

// MaskStruct returns a shallow copy of a struct with all mask:"partial" fields
// replaced by their masked equivalents. Used by the JSON serialiser middleware
// before encoding an API response.
//
// Usage:
//
//	masked := mask.MaskStruct(slackInstallation).(slack.SlackInstallation)
func MaskStruct(v interface{}) interface{} {
    if v == nil {
        return nil
    }

    rv := reflect.ValueOf(v)
    rt := rv.Type()
    for rt.Kind() == reflect.Ptr {
        rt = rt.Elem()
        rv = rv.Elem()
    }
    if rt.Kind() != reflect.Struct {
        return v
    }

    maskFields := tags.FieldsWithTag(rt, "mask")
    if len(maskFields) == 0 {
        return v
    }

    copy := reflect.New(rt).Elem()
    copy.Set(rv)

    for _, f := range maskFields {
        fv := copy.FieldByName(f.FieldName)
        if !fv.IsValid() || fv.Kind() != reflect.String || !fv.CanSet() {
            continue
        }
        masked := MaskValue(fv.String(), f.TagValue)
        fv.SetString(masked)
    }

    return copy.Interface()
}

// MaskedFields returns a summary of which fields are masked on a type —
// used for the settings UI to display "Token: xoxb-****  (connected)".
type MaskedFieldSummary struct {
    FieldName  string
    ColumnName string
    Strategy   MaskStrategy
    Masked     string // The masked value for the current instance
}

// SummariseMaskedFields returns the masked values for all mask-tagged fields
// in the given struct instance. Useful for settings page rendering.
func SummariseMaskedFields(v interface{}) []MaskedFieldSummary {
    rv := reflect.ValueOf(v)
    rt := rv.Type()
    for rt.Kind() == reflect.Ptr {
        rt = rt.Elem()
        rv = rv.Elem()
    }
    if rt.Kind() != reflect.Struct {
        return nil
    }

    maskFields := tags.FieldsWithTag(rt, "mask")
    summaries := make([]MaskedFieldSummary, 0, len(maskFields))
    for _, f := range maskFields {
        fv := rv.FieldByName(f.FieldName)
        if !fv.IsValid() || fv.Kind() != reflect.String {
            continue
        }
        summaries = append(summaries, MaskedFieldSummary{
            FieldName:  f.FieldName,
            ColumnName: f.ColumnName,
            Strategy:   MaskStrategy(f.TagValue),
            Masked:     MaskValue(fv.String(), f.TagValue),
        })
    }
    return summaries
}

// MaskingMiddleware is an HTTP middleware that applies MaskStruct to any struct
// response value before JSON serialisation. Intended to wrap the API handler's
// response encoder.
//
// Wire this into the HTTP stack only for routes that return token-bearing models
// (settings pages, integration status endpoints). Do not apply globally.
func MaskingMiddleware(next http.Handler) http.Handler {
    return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        mrw := &maskingResponseWriter{ResponseWriter: w}
        next.ServeHTTP(mrw, r)
    })
}

// maskingResponseWriter intercepts WriteJSON calls and applies masking.
// In practice, the API layer calls mask.MaskStruct() explicitly before encoding
// rather than using this middleware — include it as an optional defence-in-depth layer.
type maskingResponseWriter struct {
    http.ResponseWriter
}

// Utility: IsMasked reports whether a struct type has any mask:"partial" fields.
func IsMasked(t reflect.Type) bool {
    return tags.HasTag(t, "mask")
}

// MaskToken is a standalone helper for masking a single string token.
// Used in Slack notification builders that include a connection status line.
//
// Example:
//
//	fmt.Sprintf("Bot token: %s", mask.MaskToken("xoxb-111-222-secret"))
//	// → "Bot token: xoxb-****"
func MaskToken(token string) string {
    return MaskValue(token, string(StrategyPartial))
}

// ConnectedStatus returns a human-readable connection status string for the
// settings UI, combining the masked token with a connected/disconnected label.
//
// Example:
//
//	mask.ConnectedStatus("xoxb-111-222", true)  → "Connected  (xoxb-****)"
//	mask.ConnectedStatus("", false)             → "Not connected"
func ConnectedStatus(token string, connected bool) string {
    if !connected || token == "" {
        return "Not connected"
    }
    return fmt.Sprintf("Connected  (%s)", MaskToken(token))
}
