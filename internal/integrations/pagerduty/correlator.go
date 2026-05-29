package pagerduty

import (
	"context"
	"log/slog"
	"time"
)

// Correlator handles linking PagerDuty incidents to deployments.
type Correlator struct {
	deployService  DeployService
	serviceMapRepo ServiceMapRepository
}

// DeployService is the interface for deployment-related operations.
type DeployService interface {
	UpdateHealthStatus(ctx context.Context, deployID uint64, status HealthStatus, incidentID, incidentURL string) error
}

// ServiceMapRepository is the interface for storing and retrieving service mappings.
type ServiceMapRepository interface {
	FindByPagerDutyService(ctx context.Context, orgID uint64, serviceName string) (*ServiceMap, error)
}

// NewCorrelator creates a new Correlator.
func NewCorrelator(deployService DeployService, serviceMapRepo ServiceMapRepository) *Correlator {
	return &Correlator{
		deployService:  deployService,
		serviceMapRepo: serviceMapRepo,
	}
}

// LinkIncidentToDeploy links a PagerDuty incident to the most recent affected deployment.
func (c *Correlator) LinkIncidentToDeploy(ctx context.Context, incident *PagerDutyIncident) error {
	// TODO: Get orgID from context or installation
	_ = uint64(0) // Placeholder
	slog.Info("linking incident to deploy", "incident_id", incident.ID, "service", incident.Service.Name)

	// Look back 2 hours to find the culprit deployment
	cutoff := incident.CreatedAt.Add(-2 * time.Hour)

	// TODO: Query the DB for the most recent deploy touching this service
	// This is a placeholder, you'll need to implement the actual DB query
	slog.Info("looking for culprit deployments", "cutoff", cutoff, "service", incident.Service.Name)

	return nil
}

// HandleResolvedIncident updates the health status of a deployment when an incident is resolved.
func (c *Correlator) HandleResolvedIncident(ctx context.Context, incident *PagerDutyIncident) error {
	slog.Info("handling resolved incident", "incident_id", incident.ID)
	// TODO: Find the deployment linked to this incident and update its health status
	return nil
}
