export type DeployStatus = "pending" | "approved" | "rejected" | "skipped" | "voided";
export type RiskLevel = "low" | "medium" | "high" | "critical";

export interface DeployEvent {
  id: string;
  org_id: string;
  source: string;
  repo_owner: string;
  repo_name: string;
  commit_sha: string;
  commit_message: string;
  branch: string;
  environment: string;
  author_login: string;
  triggered_at: string;
  completed_at?: string;
  risk_score: number;
  risk_level: RiskLevel;
  status: DeployStatus;
  files_changed: number;
  additions: number;
  deletions: number;
  deploy_log_url?: string;
}

export interface Approval {
  id: string;
  deploy_event_id: string;
  status: "pending" | "granted" | "rejected" | "expired";
  approver_github_login?: string;
  decided_at?: string;
  comment?: string;
}

export type ComplianceFramework = "soc2" | "hipaa" | "iso27001";
export type EvidenceReportStatus = "pending" | "generating" | "ready" | "failed";

export interface EvidenceReport {
  id: string;
  org_id: string;
  framework: ComplianceFramework;
  period_start: string;
  period_end: string;
  status: EvidenceReportStatus;
  download_url?: string;
  expires_at?: string;
  summary?: {
    total_deploys: number;
    approved_pct: number;
    exceptions: number;
  };
}

export interface DORASnapshot {
  id: string;
  org_id: string;
  period_start: string;
  period_end: string;
  deployment_frequency: number;
  deployment_frequency_tier: string;
  lead_time_hours: number;
  lead_time_tier: string;
  mttr_hours: number;
  mttr_tier: string;
  change_failure_rate: number;
  change_failure_tier: string;
  total_deploys: number;
  total_incidents: number;
}
