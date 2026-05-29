package audit

import (
	"context"
	"time"

	"gorm.io/gorm"
)

type auditRepository struct {
	db *gorm.DB
}

// NewRepository creates a new audit repository
func NewRepository(db *gorm.DB) Repository {
	return &auditRepository{db: db}
}

// AuditExportRow is the flat struct used for compliance export queries.
// No GORM model — produced directly from raw SQL.
type AuditExportRow struct {
	OccurredAt        time.Time  `db:"occurred_at"`
	EventType         string     `db:"event_type"`
	ResourceType      string     `db:"resource_type"`
	ResourceID        string     `db:"resource_id"`
	ActorLogin        string     `db:"actor_login"`
	ApproverLogin     string     `db:"approver_login"`
	ApproverDecidedAt *time.Time `db:"approver_decided_at"`
	DeployEnvironment string     `db:"deploy_environment"`
	DeployRiskLevel   string     `db:"deploy_risk_level"`
	TicketKey         string     `db:"ticket_key"`
	RecordHash        string     `db:"record_hash"`
}

func (r *auditRepository) Create(ctx context.Context, record *AuditRecord) error {
	return r.db.WithContext(ctx).Create(record).Error
}

func (r *auditRepository) FindByID(ctx context.Context, orgID, id uint64) (*AuditRecord, error) {
	var record AuditRecord
	err := r.db.WithContext(ctx).Where("org_id = ? AND id = ?", orgID, id).First(&record).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, ErrAuditRecordNotFound
		}
		return nil, err
	}
	return &record, nil
}

func (r *auditRepository) ListByOrg(ctx context.Context, orgID uint64, limit, offset int) ([]*AuditRecord, error) {
	var records []*AuditRecord
	err := r.db.WithContext(ctx).Where("org_id = ?", orgID).
		Order("occurred_at DESC").
		Limit(limit).
		Offset(offset).
		Find(&records).Error
	return records, err
}

// ListForExport fetches the full audit trail for a compliance export.
// Joins audit_records with deploy_events, approvals, users, and linked_tickets
// in a single query — efficient for generating the evidence PDF.
func (r *auditRepository) ListForExport(
	ctx context.Context,
	orgID uint64,
	from, to time.Time,
) ([]AuditExportRow, error) {
	var rows []AuditExportRow
	result := r.db.WithContext(ctx).Raw(`
		SELECT
		    ar.occurred_at,
		    ar.event_type,
		    ar.resource_type,
		    ar.resource_id,
		    COALESCE(u.display_name, ar.actor_service, 'system')  AS actor_login,
		    COALESCE(approver.display_name, '')                    AS approver_login,
		    a.decided_at                                           AS approver_decided_at,
		    de.environment                                         AS deploy_environment,
		    de.risk_level                                          AS deploy_risk_level,
		    COALESCE(lt.ticket_key, '')                            AS ticket_key,
		    ar.record_hash
		FROM audit_records ar
		LEFT JOIN users            u        ON u.id = ar.actor_user_id
		LEFT JOIN deploy_events    de       ON de.id::text = ar.resource_id
		                                   AND ar.resource_type = 'deploy_event'
		LEFT JOIN approvals        a        ON a.deploy_event_id = de.id
		LEFT JOIN users            approver ON approver.id = a.approver_user_id
		LEFT JOIN linked_tickets   lt       ON lt.deploy_event_id = de.id
		                                   AND lt.deleted_at IS NULL
		WHERE ar.org_id = ?
		  AND ar.occurred_at BETWEEN ? AND ?
		ORDER BY ar.occurred_at ASC
	`, orgID, from, to).Scan(&rows)

	return rows, result.Error
}

// VerifyChain walks the hash chain for an org and returns the first broken link.
func (r *auditRepository) VerifyChain(
	ctx context.Context,
	orgID uint64,
) (broken bool, brokenAtID uint64, err error) {
	var records []struct {
		ID           uint64  `db:"id"`
		RecordHash   string  `db:"record_hash"`
		PreviousHash *string `db:"previous_hash"`
	}

	result := r.db.WithContext(ctx).Raw(`
		SELECT id, record_hash, previous_hash
		FROM audit_records
		WHERE org_id = ?
		ORDER BY created_at ASC
	`, orgID).Scan(&records)
	if result.Error != nil {
		return false, 0, result.Error
	}

	for i := 1; i < len(records); i++ {
		cur := records[i]
		prev := records[i-1]
		if cur.PreviousHash == nil || *cur.PreviousHash != prev.RecordHash {
			return true, cur.ID, nil
		}
	}
	return false, 0, nil
}
