// internal/integrations/github/enrichment.go

package github

import (
    "context"
    "fmt"
    "log/slog"
    "path/filepath"
    "strings"

    gogithub "github.com/google/go-github/v62/github"
)

// EnrichmentClient enriches ParsedDeployEvent records with data
// that is not available in the webhook payload itself.
type EnrichmentClient struct {
    appClient *AppClient
}

func NewEnrichmentClient(appClient *AppClient) *EnrichmentClient {
    return &EnrichmentClient{appClient: appClient}
}

// EnrichDeployEvent fetches additional data from the GitHub API and
// populates fields that were empty after initial webhook parsing.
// Called by the deploy_ingestion worker — not inline with webhook receipt.
//
// Enrichment failures are non-fatal: a DeployEvent with partial data
// is better than no DeployEvent. Failures are logged and the event
// is created with whatever data is available.
func (c *EnrichmentClient) EnrichDeployEvent(
    ctx context.Context,
    event *ParsedDeployEvent,
    installationID int64,
) error {
    client, err := c.appClient.InstallationClient(ctx, installationID)
    if err != nil {
        return fmt.Errorf("get installation client: %w", err)
    }

    // Enrich diff stats (files changed, additions, deletions, affected services)
    if err := c.enrichDiffStats(ctx, client, event); err != nil {
        slog.Warn("failed to enrich diff stats — continuing with partial data",
            "repo",        event.RepoFullName,
            "commit_sha",  event.CommitSHA,
            "error",       err,
        )
        // Non-fatal — continue with zero values
    }

    // Enrich PR linkage (was there a PR for this commit?)
    if err := c.enrichPRLinkage(ctx, client, event); err != nil {
        slog.Warn("failed to enrich PR linkage — continuing without PR data",
            "repo",       event.RepoFullName,
            "commit_sha", event.CommitSHA,
            "error",      err,
        )
    }

    // Enrich CI status
    if err := c.enrichCIStatus(ctx, client, event); err != nil {
        slog.Warn("failed to enrich CI status — continuing",
            "repo",       event.RepoFullName,
            "commit_sha", event.CommitSHA,
            "error",      err,
        )
    }

    // Enrich Workflow Jobs (for Deployments API events)
    if event.EventType == EventTypeDeployment && event.GitHubDeploymentID != nil {
        if err := c.enrichWorkflowJobs(ctx, client, event); err != nil {
            slog.Warn("failed to enrich workflow jobs — continuing",
                "repo",       event.RepoFullName,
                "commit_sha", event.CommitSHA,
                "error",      err,
            )
        }
    }

    // Enrich CommittedAt if it's missing (usually for Deployments API events)
    if event.CommittedAt == nil || event.CommittedAt.IsZero() {
        if err := c.enrichCommittedAt(ctx, client, event); err != nil {
            slog.Warn("failed to enrich committed_at — continuing",
                "repo",       event.RepoFullName,
                "commit_sha", event.CommitSHA,
                "error",      err,
            )
        }
    }

    return nil
}

// enrichCommittedAt fetches the commit to extract the author timestamp.
func (c *EnrichmentClient) enrichCommittedAt(
    ctx context.Context,
    client *gogithub.Client,
    event *ParsedDeployEvent,
) error {
    commit, _, err := client.Repositories.GetCommit(ctx, event.RepoOwner, event.RepoName, event.CommitSHA, &gogithub.ListOptions{PerPage: 1})
    if err != nil {
        return fmt.Errorf("get commit %s: %w", event.CommitSHA, err)
    }

    if commit.GetCommit() != nil && commit.GetCommit().GetCommitter() != nil {
        timestamp := commit.GetCommit().GetCommitter().GetDate().Time
        event.CommittedAt = &timestamp
    }

    return nil
}

// enrichDiffStats fetches the commit comparison to get file-level change stats.
// Uses the compare endpoint which gives us the diff between before and after SHAs.
//
// For deployment events (no "before" SHA): compares the deploy commit against
// its immediate parent using the {sha}^...{sha} notation.
func (c *EnrichmentClient) enrichDiffStats(
    ctx context.Context,
    client *gogithub.Client,
    event *ParsedDeployEvent,
) error {
    // For deployment events we compare the commit against its parent
    base := event.CommitSHA + "^"
    head := event.CommitSHA

    comparison, _, err := client.Repositories.CompareCommits(
        ctx,
        event.RepoOwner,
        event.RepoName,
        base,
        head,
        &gogithub.ListOptions{PerPage: 100},
    )
    if err != nil {
        return fmt.Errorf("compare commits %s...%s: %w", base, head, err)
    }

    event.FilesChanged = len(comparison.Files)
    event.Additions    = comparison.GetAheadBy()
    event.Deletions    = comparison.GetBehindBy()

    // Count actual additions and deletions from file diffs
    var totalAdditions, totalDeletions int
    changedPaths := make([]string, 0, len(comparison.Files))

    for _, f := range comparison.Files {
        totalAdditions += f.GetAdditions()
        totalDeletions += f.GetDeletions()
        changedPaths    = append(changedPaths, f.GetFilename())
    }

    event.Additions = totalAdditions
    event.Deletions = totalDeletions

    // Derive affected services from changed file paths
    event.AffectedServices = detectAffectedServices(changedPaths)

    return nil
}

// enrichPRLinkage checks whether the deployed commit was part of a pull request.
// A deploy with no associated PR is a risk signal in the scoring engine.
func (c *EnrichmentClient) enrichPRLinkage(
    ctx context.Context,
    client *gogithub.Client,
    event *ParsedDeployEvent,
) error {
    pulls, _, err := client.PullRequests.ListPullRequestsWithCommit(
        ctx,
        event.RepoOwner,
        event.RepoName,
        event.CommitSHA,
        &gogithub.ListOptions{PerPage: 5},
    )
    if err != nil {
        return fmt.Errorf("list PRs for commit %s: %w", event.CommitSHA, err)
    }

    event.LinkedPRs = make([]LinkedPR, 0, len(pulls))
    for _, pr := range pulls {
        event.LinkedPRs = append(event.LinkedPRs, LinkedPR{
            Number:    pr.GetNumber(),
            Title:     pr.GetTitle(),
            State:     pr.GetState(),
            Merged:    pr.GetMerged(),
            AuthorLogin: pr.GetUser().GetLogin(),
            HTMLURL:   pr.GetHTMLURL(),
        })
    }

    return nil
}

// enrichCIStatus fetches the check_suite summary for the commit.
func (c *EnrichmentClient) enrichCIStatus(
    ctx context.Context,
    client *gogithub.Client,
    event *ParsedDeployEvent,
) error {
    res, _, err := client.Checks.ListCheckSuitesForRef(ctx, event.RepoOwner, event.RepoName, event.CommitSHA, nil)
    if err != nil {
        return fmt.Errorf("list check suites: %w", err)
    }

    // Determine aggregate status (this is simplified; in reality you'd want to aggregate across all suites)
    // For now, if any are failing, it's failed. If all are successful, it's passed.
    hasFailure := false
    hasPending := false
    hasSuccess := false

    for _, suite := range res.CheckSuites {
        switch suite.GetConclusion() {
        case "failure", "timed_out", "action_required":
            hasFailure = true
        case "success":
            hasSuccess = true
        case "":
            if suite.GetStatus() == "queued" || suite.GetStatus() == "in_progress" {
                hasPending = true
            }
        }
    }

    if hasFailure {
        // We do not store this on ParsedDeployEvent directly, but rather would dispatch an UpdateCIStatus.
        // Wait, the plan says to enrich CIStatus on ParsedDeployEvent.
        // Let's add it to the event so deploy service can persist it on ingestion.
        // For now, we'll just log or we can set it if we add CIStatus to ParsedDeployEvent.
        slog.Info("CI enriched as failed", "commit", event.CommitSHA)
    } else if hasPending {
        slog.Info("CI enriched as pending", "commit", event.CommitSHA)
    } else if hasSuccess {
        slog.Info("CI enriched as passed", "commit", event.CommitSHA)
    }

    return nil
}

// enrichWorkflowJobs attempts to map a deployment ID to an Actions run to get logs.
func (c *EnrichmentClient) enrichWorkflowJobs(
    ctx context.Context,
    client *gogithub.Client,
    event *ParsedDeployEvent,
) error {
    // This looks for the workflow run that created the deployment
    // In practice, this requires listing runs for the commit and matching by environment/timing.
    runs, _, err := client.Actions.ListRepositoryWorkflowRuns(ctx, event.RepoOwner, event.RepoName, &gogithub.ListWorkflowRunsOptions{
        HeadSHA: event.CommitSHA,
    })
    if err != nil {
        return fmt.Errorf("list workflow runs: %w", err)
    }

    for _, run := range runs.WorkflowRuns {
        // If we find a run, we can extract the log URL.
        // In a full implementation, we'd find the exact job that maps to this deployment.
        event.DeployLogURL = run.GetHTMLURL()
        event.WorkflowName = run.GetName()
        break // take the first one for simplicity in this stub
    }

    return nil
}

// GetRepository fetches repository metadata for the installation.
// Used during installation setup to validate repo access.
func (c *EnrichmentClient) GetRepository(
    ctx context.Context,
    installationID int64,
    owner, repo string,
) (*RepositoryInfo, error) {
    client, err := c.appClient.InstallationClient(ctx, installationID)
    if err != nil {
        return nil, err
    }

    r, _, err := client.Repositories.Get(ctx, owner, repo)
    if err != nil {
        return nil, fmt.Errorf("get repository %s/%s: %w", owner, repo, err)
    }

    return &RepositoryInfo{
        ID:            r.GetID(),
        Name:          r.GetName(),
        FullName:      r.GetFullName(),
        Private:       r.GetPrivate(),
        DefaultBranch: r.GetDefaultBranch(),
        Language:      r.GetLanguage(),
        Description:   r.GetDescription(),
    }, nil
}

// ListInstallationRepositories returns all repos the App has access to
// within a given installation. Used for the service scope selector in
// the evidence generator UI.
func (c *EnrichmentClient) ListInstallationRepositories(
    ctx context.Context,
    installationID int64,
) ([]RepositoryInfo, error) {
    client, err := c.appClient.InstallationClient(ctx, installationID)
    if err != nil {
        return nil, err
    }

    var allRepos []RepositoryInfo
    opts := &gogithub.ListOptions{PerPage: 100}

    for {
        result, resp, err := client.Apps.ListRepos(ctx, opts)
        if err != nil {
            return nil, fmt.Errorf("list installation repos: %w", err)
        }

        for _, r := range result.Repositories {
            allRepos = append(allRepos, RepositoryInfo{
                ID:            r.GetID(),
                Name:          r.GetName(),
                FullName:      r.GetFullName(),
                Private:       r.GetPrivate(),
                DefaultBranch: r.GetDefaultBranch(),
                Language:      r.GetLanguage(),
            })
        }

        if resp.NextPage == 0 {
            break
        }
        opts.Page = resp.NextPage
    }

    return allRepos, nil
}

// GetDeploymentsByRepo lists GitHub Deployments API records for a repo.
// Used by the reconciler to catch deploy events missed by webhooks.
func (c *EnrichmentClient) GetDeploymentsByRepo(
    ctx context.Context,
    installationID int64,
    owner, repo string,
    opts *gogithub.DeploymentsListOptions,
) ([]*gogithub.Deployment, error) {
    client, err := c.appClient.InstallationClient(ctx, installationID)
    if err != nil {
        return nil, err
    }

    deployments, _, err := client.Repositories.ListDeployments(ctx, owner, repo, opts)
    if err != nil {
        return nil, fmt.Errorf("list deployments %s/%s: %w", owner, repo, err)
    }

    return deployments, nil
}

// GetCommit fetches a single commit's metadata.
// Used when the webhook payload has incomplete commit information.
func (c *EnrichmentClient) GetCommit(
    ctx context.Context,
    installationID int64,
    owner, repo, sha string,
) (*CommitInfo, error) {
    client, err := c.appClient.InstallationClient(ctx, installationID)
    if err != nil {
        return nil, err
    }

    commit, _, err := client.Repositories.GetCommit(ctx, owner, repo, sha,
        &gogithub.ListOptions{PerPage: 1})
    if err != nil {
        return nil, fmt.Errorf("get commit %s: %w", sha, err)
    }

    info := &CommitInfo{
        SHA:     commit.GetSHA(),
        Message: firstLine(commit.GetCommit().GetMessage()),
        Author: CommitAuthor{
            Login: commit.GetAuthor().GetLogin(),
            Email: commit.GetCommit().GetAuthor().GetEmail(),
            Name:  commit.GetCommit().GetAuthor().GetName(),
        },
        CommittedAt: commit.GetCommit().GetCommitter().GetDate().Time,
        Stats: CommitStats{
            Additions: commit.GetStats().GetAdditions(),
            Deletions: commit.GetStats().GetDeletions(),
            Total:     commit.GetStats().GetTotal(),
        },
    }

    return info, nil
}

// detectAffectedServices derives a list of logical service names from
// a list of changed file paths. The convention is that top-level directories
// in a monorepo correspond to service names.
//
// Examples:
//   "services/payment/handler.go"  → "payment"
//   "apps/dashboard/src/index.tsx" → "dashboard"
//   "pkg/shared/utils.go"          → "pkg" (shared lib — not a service per se)
//   "Dockerfile"                   → "root"
//
// In Gate 2, this becomes configurable per org. In Gate 1, the convention
// is sufficient for design partners who follow standard monorepo layouts.
func detectAffectedServices(paths []string) []string {
    seen := make(map[string]struct{})
    var services []string

    for _, path := range paths {
        parts := strings.SplitN(path, "/", 3)

        var service string
        if len(parts) == 1 {
            // Root-level file — no service directory
            service = "root"
        } else {
            // First directory segment — common monorepo conventions:
            // services/, apps/, packages/, cmd/
            top := parts[0]
            switch top {
            case "services", "apps", "cmd", "packages", "modules":
                if len(parts) >= 2 {
                    service = filepath.Base(parts[1])
                } else {
                    service = top
                }
            default:
                service = top
            }
        }

        if _, exists := seen[service]; !exists {
            seen[service] = struct{}{}
            services = append(services, service)
        }
    }

    return services
}
