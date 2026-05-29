
package pagerduty

// Service provides PagerDuty integration functionality.
type Service struct {
	webhookHandler *WebhookHandler
	correlator     *Correlator
}

// NewService creates a new PagerDuty service.
func NewService(webhookHandler *WebhookHandler, correlator *Correlator) *Service {
	return &Service{
		webhookHandler: webhookHandler,
		correlator:     correlator,
	}
}

// WebhookHandler returns the HTTP handler for PagerDuty webhooks.
func (s *Service) WebhookHandler() *WebhookHandler {
	return s.webhookHandler
}
