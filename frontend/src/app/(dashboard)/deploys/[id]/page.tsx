"use client";

import { useState } from "react";
import { useParams, useRouter } from "next/navigation";
import { format } from "date-fns";
import { useQuery, useMutation, useQueryClient } from "@tanstack/react-query";
import { useApi } from "@/lib/useApi";
import { RiskBadge, StatusBadge } from "@/components/deploy-status-badge";
import { ArrowLeft, CheckCircle, XCircle } from "@phosphor-icons/react";

export default function DeployDetailPage() {
  const params = useParams();
  const router = useRouter();
  const queryClient = useQueryClient();
  const { getBoundApi, hasOrg } = useApi();
  const [comment, setComment] = useState("");
  const deployId = params.id as string;

  const { data: deploy, isLoading, error } = useQuery({
    queryKey: ["deploys", "detail", deployId],
    queryFn: async () => {
      const boundApi = await getBoundApi();
      return await boundApi.deploys.get(deployId);
    },
    enabled: hasOrg,
  });

  const mutation = useMutation({
    mutationFn: async ({ action }: { action: "approve" | "reject" }) => {
      const boundApi = await getBoundApi();
      if (action === "approve") {
        return await boundApi.approvals.approve(deployId, comment);
      } else {
        return await boundApi.approvals.reject(deployId, comment);
      }
    },
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ["deploys"] });
      router.refresh();
    },
  });

  if (!hasOrg) return <div className="p-8">Please select an organization.</div>;
  if (isLoading) return <div className="p-8">Loading details...</div>;
  if (error) return <div className="p-8 text-red-500">Error: {(error as Error).message}</div>;
  if (!deploy) return <div className="p-8">Deploy not found</div>;

  return (
    <div className="flex flex-col gap-6 p-8 max-w-4xl">
      <div>
        <button
          onClick={() => router.back()}
          className="flex items-center gap-2 text-sm text-muted-foreground hover:text-foreground mb-4 transition-colors"
        >
          <ArrowLeft className="size-4" />
          Back to Deployments
        </button>
        <div className="flex items-start justify-between">
          <div>
            <h1 className="text-2xl font-bold tracking-tight mb-2">Deploy {deploy.commit_sha.substring(0, 7)}</h1>
            <div className="flex gap-3 items-center text-sm text-muted-foreground">
              <span>{deploy.repo_owner}/{deploy.repo_name}</span>
              <span>•</span>
              <span>{deploy.branch}</span>
              <span>•</span>
              <span>by {deploy.author_login}</span>
            </div>
          </div>
          <div className="flex gap-2">
            <RiskBadge level={deploy.risk_level} />
            <StatusBadge status={deploy.status} />
          </div>
        </div>
      </div>

      <div className="grid grid-cols-3 gap-6">
        <div className="col-span-2 flex flex-col gap-6">
          <div className="rounded-xl border border-border bg-card shadow-sm p-6">
            <h2 className="text-lg font-semibold mb-4">Commit Details</h2>
            <div className="font-mono text-sm bg-muted p-4 rounded-md mb-4 whitespace-pre-wrap">
              {deploy.commit_message}
            </div>
            <div className="flex gap-6 text-sm">
              <div className="flex flex-col">
                <span className="text-muted-foreground">Files Changed</span>
                <span className="font-medium">{deploy.files_changed}</span>
              </div>
              <div className="flex flex-col">
                <span className="text-muted-foreground">Additions</span>
                <span className="font-medium text-emerald-500">+{deploy.additions}</span>
              </div>
              <div className="flex flex-col">
                <span className="text-muted-foreground">Deletions</span>
                <span className="font-medium text-red-500">-{deploy.deletions}</span>
              </div>
            </div>
          </div>

          {deploy.status === "pending" && (
            <div className="rounded-xl border border-border bg-card shadow-sm p-6">
              <h2 className="text-lg font-semibold mb-4">Compliance Approval</h2>
              <p className="text-sm text-muted-foreground mb-4">
                This deployment has been flagged as <strong className="text-orange-500 uppercase">{deploy.risk_level}</strong> risk.
                Approval is required before it can proceed to production.
              </p>
              
              <textarea
                value={comment}
                onChange={(e) => setComment(e.target.value)}
                placeholder="Add an optional comment for the audit log..."
                className="w-full min-h-[100px] p-3 rounded-md border border-input bg-background text-sm mb-4"
              />
              
              <div className="flex gap-3">
                <button
                  onClick={() => mutation.mutate({ action: "approve" })}
                  disabled={mutation.isPending}
                  className="flex flex-1 items-center justify-center gap-2 rounded-md bg-emerald-600 px-4 py-2 text-sm font-medium text-white hover:bg-emerald-700 disabled:opacity-50 transition-colors"
                >
                  <CheckCircle className="size-5" />
                  Approve Deployment
                </button>
                <button
                  onClick={() => mutation.mutate({ action: "reject" })}
                  disabled={mutation.isPending}
                  className="flex flex-1 items-center justify-center gap-2 rounded-md border border-red-800 bg-red-950/30 px-4 py-2 text-sm font-medium text-red-500 hover:bg-red-950/50 disabled:opacity-50 transition-colors"
                >
                  <XCircle className="size-5" />
                  Reject
                </button>
              </div>
            </div>
          )}
        </div>

        <div className="flex flex-col gap-6">
          <div className="rounded-xl border border-border bg-card shadow-sm p-6">
            <h2 className="text-sm font-semibold text-muted-foreground uppercase tracking-wider mb-4">Timeline</h2>
            <div className="flex flex-col gap-4 relative before:absolute before:inset-y-0 before:left-[11px] before:w-[2px] before:bg-border">
              <div className="flex gap-4 relative">
                <div className="size-6 rounded-full bg-primary ring-4 ring-card flex items-center justify-center shrink-0 z-10" />
                <div className="flex flex-col pb-4">
                  <span className="text-sm font-medium">Triggered</span>
                  <span className="text-xs text-muted-foreground">
                    {format(new Date(deploy.triggered_at), "MMM d, yyyy HH:mm:ss")}
                  </span>
                </div>
              </div>
              
              {deploy.completed_at && (
                <div className="flex gap-4 relative">
                  <div className="size-6 rounded-full bg-border ring-4 ring-card flex items-center justify-center shrink-0 z-10" />
                  <div className="flex flex-col">
                    <span className="text-sm font-medium">Completed</span>
                    <span className="text-xs text-muted-foreground">
                      {format(new Date(deploy.completed_at), "MMM d, yyyy HH:mm:ss")}
                    </span>
                  </div>
                </div>
              )}
            </div>
          </div>
        </div>
      </div>
    </div>
  );
}
