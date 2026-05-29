package pii

import (
	"context"
	"fmt"
	"reflect"
	"time"

	"gorm.io/gorm"

	"github.com/samaasi/watchnoc/internal/platform/tags"
)

// PIINullificationDays is the number of days after soft-delete at which PII
// fields are nullified. Satisfies GDPR Art. 17 / CCPA § 1798.105.
const PIINullificationDays = 90

// NullifyPIIFields sets all pii:"true" fields to NULL on every record in the
// given table whose deleted_at is older than PIINullificationDays days and
// whose PII has not already been nullified.
//
// Called by the nightly retention worker. Safe to call multiple times —
// already-nullified rows are skipped by the WHERE clause.
//
// tableName:    SQL table name (e.g. "users")
// modelType:    reflect.Type of the GORM model (e.g. reflect.TypeOf(User{}))
// piiColumn:    One representative PII column — used as a sentinel to detect
//
//	already-nullified rows (WHERE <piiColumn> IS NOT NULL)
func NullifyPIIFields(
	ctx context.Context,
	db *gorm.DB,
	tableName string,
	modelType reflect.Type,
	sentinelColumn string,
	softDeleteCol string,
) (int64, error) {
	piiFields := tags.FieldsWithTag(modelType, "pii")
	if len(piiFields) == 0 {
		return 0, nil
	}

	if softDeleteCol == "" {
		// Physical hard-purge table — rows are deleted, not soft-deleted.
		// No need to nullify.
		return 0, nil
	}

	// Build: SET col1 = NULL, col2 = NULL, ...
	setClauses := make([]string, 0, len(piiFields))
	for _, f := range piiFields {
		if f.TagValue != "true" {
			continue
		}
		setClauses = append(setClauses, fmt.Sprintf("%s = NULL", f.ColumnName))
	}
	if len(setClauses) == 0 {
		return 0, nil
	}

	setClauses = append(setClauses, "updated_at = NOW()")
	setSQL := joinComma(setClauses)

	cutoff := time.Now().UTC().AddDate(0, 0, -PIINullificationDays)

	result := db.WithContext(ctx).Exec(
		fmt.Sprintf(`
            UPDATE %s
            SET %s
            WHERE %s IS NOT NULL
              AND %s < ?
              AND %s IS NOT NULL
        `, tableName, setSQL, softDeleteCol, softDeleteCol, sentinelColumn),
		cutoff,
	)

	if result.Error != nil {
		return 0, fmt.Errorf("nullify PII on %s: %w", tableName, result.Error)
	}

	return result.RowsAffected, nil
}

// PIINullificationTargets lists all tables and their sentinel columns that the
// retention worker should process for PII nullification.
// Add a new entry here whenever a model gains a pii:"true" field.
var PIINullificationTargets = []NullificationTarget{
	{Table: "users", SentinelColumn: "email", ModelType: reflect.TypeOf(userModel{})},
	{Table: "deploy_events", SentinelColumn: "author_login", ModelType: reflect.TypeOf(deployEventModel{})},
	{Table: "approvals", SentinelColumn: "approver_github_login", ModelType: reflect.TypeOf(approvalModel{})},
	{Table: "audit_records", SentinelColumn: "ip_address", ModelType: reflect.TypeOf(auditRecordModel{})},
	{Table: "github_installations", SentinelColumn: "installer_github_login", ModelType: reflect.TypeOf(githubInstallModel{})},
	{Table: "slack_installations", SentinelColumn: "bot_token_encrypted", ModelType: reflect.TypeOf(slackInstallModel{})},
	{Table: "gitlab_installations", SentinelColumn: "access_token_encrypted", ModelType: reflect.TypeOf(gitlabInstallModel{})},
	{Table: "sso_configs", SentinelColumn: "config", ModelType: reflect.TypeOf(ssoConfigModel{})},
}

type NullificationTarget struct {
	Table          string
	SentinelColumn string
	ModelType      reflect.Type
}

// Model stubs used only for reflect.TypeOf — they mirror the pii:"true" tags
// from the real domain models. Kept here so this package does not import domain
// packages (avoiding import cycles).
type userModel struct {
	Email       string `pii:"true"`
	DisplayName string `pii:"true"`
	AvatarURL   string `pii:"true"`
}

type deployEventModel struct {
	AuthorLogin string `pii:"true"`
	AuthorEmail string `pii:"true"`
}

type approvalModel struct {
	ApproverGitHubLogin string `pii:"true"`
}

type auditRecordModel struct {
	IPAddress string `pii:"true"`
}

type githubInstallModel struct {
	InstallerGitHubLogin string `pii:"true"`
}

type slackInstallModel struct {
	BotTokenEncrypted string `pii:"true" crypto:"true"`
}

type gitlabInstallModel struct {
	AccessTokenEncrypted   string `pii:"true" crypto:"true"`
	RefreshTokenEncrypted  string `pii:"true" crypto:"true"`
	WebhookSecretEncrypted string `pii:"true" crypto:"true"`
}

type ssoConfigModel struct {
	Config string `pii:"true" crypto:"true"`
}

func joinComma(s []string) string {
	out := ""
	for i, v := range s {
		if i > 0 {
			out += ", "
		}
		out += v
	}
	return out
}

// CollectPIIColumns returns the database column names of all fields tagged with `pii:"true"`.
func CollectPIIColumns(t reflect.Type) []string {
	fields := tags.FieldsWithTag(t, "pii")
	var columns []string
	for _, f := range fields {
		if f.TagValue == "true" {
			columns = append(columns, f.ColumnName)
		}
	}
	return columns
}
