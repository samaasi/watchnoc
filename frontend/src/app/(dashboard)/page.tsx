"use client";

import Link from "next/link";
import { formatDistanceToNow } from "date-fns";
import { useQuery } from "@tanstack/react-query";
import { useApi } from "@/lib/useApi";
import { RiskBadge, StatusBadge } from "@/components/deploy-status-badge";
import { GitCommit, GitBranch, GithubLogo } from "@phosphor-icons/react";
import { useUIStore } from "@/lib/store";

export default function DeploymentsPage() {
  const { getBoundApi, hasOrg } = useApi();
  const isSidebarCollapsed = useUIStore((state) => state.isSidebarCollapsed); // Example usage of zustand

  const { data: deploys = [], isLoading, error } = useQuery({
    queryKey: ["deploys", "list"],
    queryFn: async () => {
      const boundApi = await getBoundApi();
      const res = await boundApi.deploys.list();
      return res.data || [];
    },
    enabled: hasOrg,
  });

  if (!hasOrg) {
    return <div className="p-8">Please select an organization from the sidebar to view deployments.</div>;
  }

  return (
    <div className="flex flex-col gap-8 p-8">
      <div>
        <h1 className="text-2xl font-bold tracking-tight">Deployments</h1>
        <p className="text-muted-foreground mt-1">
          Monitor and approve production changes to maintain SOC 2 compliance.
        </p>
      </div>

      <div className="rounded-xl border border-border bg-card shadow-sm overflow-hidden">
        <table className="w-full text-sm text-left">
          <thead className="bg-muted/50 text-muted-foreground border-b border-border">
            <tr>
              <th className="px-6 py-4 font-medium">Commit</th>
              <th className="px-6 py-4 font-medium">Environment</th>
              <th className="px-6 py-4 font-medium">Risk Profile</th>
              <th className="px-6 py-4 font-medium">Status</th>
              <th className="px-6 py-4 font-medium">Triggered</th>
              <th className="px-6 py-4 font-medium text-right">Action</th>
            </tr>
          </thead>
          <tbody className="divide-y divide-border">
            {isLoading ? (
              <tr>
                <td colSpan={6} className="px-6 py-8 text-center text-muted-foreground">
                  Loading deployments...
                </td>
              </tr>
            ) : error ? (
              <tr>
                <td colSpan={6} className="px-6 py-8 text-center text-red-500">
                  Failed to load deployments: {(error as Error).message}
                </td>
              </tr>
            ) : deploys.length === 0 ? (
              <tr>
                <td colSpan={6} className="px-6 py-8 text-center text-muted-foreground">
                  No deployments found.
                </td>
              </tr>
            ) : (
              deploys.map((deploy) => (
                <tr key={deploy.id} className="group hover:bg-muted/30 transition-colors">
                  <td className="px-6 py-4">
                    <div className="flex flex-col gap-1.5">
                      <div className="flex items-center gap-2 font-medium">
                        <GithubLogo className="size-4 text-muted-foreground" />
                        <span className="truncate max-w-[200px]" title={deploy.commit_message}>
                          {deploy.commit_message}
                        </span>
                      </div>
                      <div className="flex items-center gap-3 text-xs text-muted-foreground font-mono">
                        <span className="flex items-center gap-1">
                          <GitCommit className="size-3" />
                          {deploy.commit_sha.substring(0, 7)}
                        </span>
                        <span className="flex items-center gap-1">
                          <GitBranch className="size-3" />
                          {deploy.branch}
                        </span>
                      </div>
                    </div>
                  </td>
                  <td className="px-6 py-4">
                    <span className="inline-flex items-center rounded-full bg-secondary px-2.5 py-0.5 text-xs font-medium text-secondary-foreground">
                      {deploy.environment}
                    </span>
                  </td>
                  <td className="px-6 py-4">
                    <RiskBadge level={deploy.risk_level} />
                  </td>
                  <td className="px-6 py-4">
                    <StatusBadge status={deploy.status} />
                  </td>
                  <td className="px-6 py-4 text-muted-foreground">
                    {formatDistanceToNow(new Date(deploy.triggered_at), { addSuffix: true })}
                  </td>
                  <td className="px-6 py-4 text-right">
                    <Link
                      href={`/deploys/${deploy.id}`}
                      className="text-sm font-medium text-primary hover:underline"
                    >
                      View Details
                    </Link>
                  </td>
                </tr>
              ))
            )}
          </tbody>
        </table>
      </div>
    </div>
  );
}
