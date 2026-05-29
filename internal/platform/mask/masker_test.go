// internal/platform/mask/masker_test.go

package mask_test

import (
    "testing"

    "github.com/stretchr/testify/assert"

    "github.com/samaasi/watchnoc/internal/platform/mask"
)

func TestMaskValue_Partial(t *testing.T) {
    tests := []struct {
        input    string
        expected string
    }{
        {"xoxb-123-456-abcdef",  "xoxb-****"},
        {"glpat-xyz789abc",      "glpat-****"},
        {"gldt_some_token",      "gldt_****"},
        {"secret.token.value",   "secret.****"},
        {"noseparator",          "no****"},
        {"ab",                   "****"}, // too short to reveal prefix
        {"",                     ""},
    }
    for _, tt := range tests {
        t.Run(tt.input, func(t *testing.T) {
            assert.Equal(t, tt.expected, mask.MaskValue(tt.input, "partial"))
        })
    }
}

func TestConnectedStatus(t *testing.T) {
    assert.Equal(t, "Connected  (xoxb-****)", mask.ConnectedStatus("xoxb-123-abc", true))
    assert.Equal(t, "Not connected", mask.ConnectedStatus("", false))
    assert.Equal(t, "Not connected", mask.ConnectedStatus("token", false))
}
