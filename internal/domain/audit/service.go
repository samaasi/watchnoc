package audit

import (
	"context"
	"encoding/json"
	"time"
)

// Repository defines the audit repository interface (extending existing)
type Repository interface {
	Create(ctx context.Context, record *AuditRecord) error
	FindByID(ctx context.Context, orgID, id uint64) (*AuditRecord, error)
	ListByOrg(ctx context.Context, orgID uint64, limit, offset int) ([]*AuditRecord, error)
	ListForExport(ctx context.Context, orgID uint64, from, to time.Time) ([]AuditExportRow, error)
	VerifyChain(ctx context.Context, orgID uint64) (bool, uint64, error)
}

// Service defines the audit service interface
type Service interface {
	LogEvent(ctx context.Context, req LogEventRequest) (uint64, error)
	GetAuditRecord(ctx context.Context, orgID, id uint64) (*AuditRecord, error)
	ListAuditRecords(ctx context.Context, orgID uint64, limit, offset int) ([]*AuditRecord, error)
	ExportAuditTrail(ctx context.Context, orgID uint64, from, to time.Time) ([]AuditExportRow, error)
	VerifyAuditChain(ctx context.Context, orgID uint64) (bool, uint64, error)
}

type LogEventRequest struct {
	OrgID          uint64
	EventType      AuditEventType
	ActorUserID    *uint64
	ActorService   string
	ResourceType   string
	ResourceID     string
	Payload        interface{}
	OccurredAt     time.Time
	TraceID        string
	IPAddress      string
}

type service struct {
	repo Repository
}

// NewService creates a new audit service
func NewService(repo Repository) Service {
	return &service{repo: repo}
}

func (s *service) LogEvent(ctx context.Context, req LogEventRequest) (uint64, error) {
	// Serialize payload to JSON
	payloadBytes, err := json.Marshal(req.Payload)
	if err != nil {
		return 0, err
	}

	record := &AuditRecord{
		OrgID:        req.OrgID,
		EventType:    req.EventType,
		ActorUserID:  req.ActorUserID,
		ActorService: req.ActorService,
		ResourceType: req.ResourceType,
		ResourceID:   req.ResourceID,
		Payload:      payloadBytes,
		OccurredAt:   req.OccurredAt,
		TraceID:      req.TraceID,
		IPAddress:    req.IPAddress,
	}

	if err := s.repo.Create(ctx, record); err != nil {
		return 0, err
	}

	return record.ID, nil
}

func (s *service) GetAuditRecord(ctx context.Context, orgID, id uint64) (*AuditRecord, error) {
	return s.repo.FindByID(ctx, orgID, id)
}

func (s *service) ListAuditRecords(ctx context.Context, orgID uint64, limit, offset int) ([]*AuditRecord, error) {
	return s.repo.ListByOrg(ctx, orgID, limit, offset)
}

func (s *service) ExportAuditTrail(ctx context.Context, orgID uint64, from, to time.Time) ([]AuditExportRow, error) {
	return s.repo.ListForExport(ctx, orgID, from, to)
}

func (s *service) VerifyAuditChain(ctx context.Context, orgID uint64) (bool, uint64, error) {
	return s.repo.VerifyChain(ctx, orgID)
}
