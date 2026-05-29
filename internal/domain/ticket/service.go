package ticket

import (
	"context"
)

type service struct {
	repo Repository
}

func NewService(repo Repository) Service {
	return &service{repo: repo}
}

func (s *service) LinkTicket(ctx context.Context, req LinkRequest) error {
	t := &LinkedTicket{
		OrgID:         req.OrgID,
		DeployEventID: req.DeployEventID,
		TicketURL:     req.TicketURL,
		TicketKey:     req.TicketKey,
		TicketSource:  req.TicketSource,
	}
	return s.repo.Save(ctx, t)
}
