package org

import (
	"context"
	"encoding/json"
)

// Service defines org service interface
type Service interface {
	CreateOrg(ctx context.Context, name, slug, plan string) (*Org, error)
	GetSettings(ctx context.Context, orgID uint64) (map[string]interface{}, error)
	UpdateSettings(ctx context.Context, orgID uint64, settings map[string]interface{}) error
	AddMember(ctx context.Context, orgID, userID uint64, role OrgMemberRole, invitedBy *uint64) error
	RemoveMember(ctx context.Context, orgID, userID uint64) error
	GetMemberRole(ctx context.Context, orgID, userID uint64) (OrgMemberRole, error)
}

type service struct {
	repo Repository
}

// NewService creates new org service
func NewService(repo Repository) Service {
	return &service{repo: repo}
}

func (s *service) CreateOrg(ctx context.Context, name, slug, plan string) (*Org, error) {
	org := &Org{
		Name:          name,
		Slug:          slug,
		Plan:          plan,
		RetentionDays: 90, // default starter plan
		Settings:      []byte("{}"),
	}
	if err := s.repo.Create(ctx, org); err != nil {
		return nil, err
	}
	return org, nil
}

func (s *service) GetSettings(ctx context.Context, orgID uint64) (map[string]interface{}, error) {
	org, err := s.repo.FindByID(ctx, orgID)
	if err != nil {
		return nil, err
	}
	// Convert datatypes.JSON (which is []byte) to map[string]interface{}
	settings := make(map[string]interface{})
	if err := json.Unmarshal(org.Settings, &settings); err != nil {
		return nil, err
	}
	return settings, nil
}

func (s *service) UpdateSettings(ctx context.Context, orgID uint64, settings map[string]interface{}) error {
	return s.repo.UpdateSettings(ctx, orgID, settings)
}

func (s *service) AddMember(ctx context.Context, orgID, userID uint64, role OrgMemberRole, invitedBy *uint64) error {
	member := &OrgMember{
		OrgID:     orgID,
		UserID:    userID,
		Role:      role,
		InvitedBy: invitedBy,
	}
	return s.repo.AddMember(ctx, member)
}

func (s *service) RemoveMember(ctx context.Context, orgID, userID uint64) error {
	return s.repo.RemoveMember(ctx, orgID, userID)
}

func (s *service) GetMemberRole(ctx context.Context, orgID, userID uint64) (OrgMemberRole, error) {
	member, err := s.repo.GetMember(ctx, orgID, userID)
	if err != nil {
		return "", err
	}
	return member.Role, nil
}
