"use client";

import { format } from "date-fns";
import { useQuery, useMutation, useQueryClient } from "@tanstack/react-query";
import { useApi } from "@/lib/useApi";
import { FilePdf, DownloadSimple, Spinner } from "@phosphor-icons/react";

export default function EvidencePage() {
  const queryClient = useQueryClient();
  const { getBoundApi, hasOrg } = useApi();

  const { data: reports = [], isLoading, error } = useQuery({
    queryKey: ["evidence", "list"],
    queryFn: async () => {
      const boundApi = await getBoundApi();
      const res = await boundApi.evidence.list();
      return res.data || [];
    },
    enabled: hasOrg,
  });

  const mutation = useMutation({
    mutationFn: async () => {
      const boundApi = await getBoundApi();
      const periodStart = new Date(new Date().setMonth(new Date().getMonth() - 1)).toISOString();
      const periodEnd = new Date().toISOString();
      return await boundApi.evidence.generate("soc2", periodStart, periodEnd);
    },
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ["evidence"] });
    },
  });

  if (!hasOrg) return <div className="p-8">Please select an organization.</div>;

  return (
    <div className="flex flex-col gap-8 p-8 max-w-5xl">
      <div className="flex justify-between items-start">
        <div>
          <h1 className="text-2xl font-bold tracking-tight">Compliance Evidence</h1>
          <p className="text-muted-foreground mt-1">
            Generate and download auditor-ready PDF reports for SOC 2 controls.
          </p>
        </div>
        <button
          onClick={() => mutation.mutate()}
          disabled={mutation.isPending}
          className="flex items-center gap-2 bg-primary text-primary-foreground px-4 py-2 rounded-md font-medium text-sm hover:bg-primary/90 transition-colors disabled:opacity-50"
        >
          {mutation.isPending ? <Spinner className="animate-spin" /> : <FilePdf />}
          Generate Report
        </button>
      </div>

      {isLoading ? (
        <div>Loading reports...</div>
      ) : error ? (
        <div className="text-red-500">Failed to load reports: {(error as Error).message}</div>
      ) : (
        <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-6">
          {reports.map((report) => (
            <div key={report.id} className="rounded-xl border border-border bg-card shadow-sm p-6 flex flex-col">
              <div className="flex items-start justify-between mb-4">
                <div className="flex items-center gap-2">
                  <FilePdf className="size-6 text-muted-foreground" />
                  <h3 className="font-semibold uppercase tracking-wider">{report.framework} Report</h3>
                </div>
                {report.status === "ready" ? (
                  <span className="inline-flex items-center rounded-full bg-emerald-500/10 px-2 py-0.5 text-xs font-medium text-emerald-500">
                    Ready
                  </span>
                ) : (
                  <span className="inline-flex items-center rounded-full bg-amber-500/10 px-2 py-0.5 text-xs font-medium text-amber-500 gap-1.5">
                    <span className="size-1.5 rounded-full bg-amber-500 animate-pulse" />
                    Generating
                  </span>
                )}
              </div>
              
              <div className="flex flex-col gap-2 text-sm text-muted-foreground mb-6 flex-1">
                <div className="flex justify-between">
                  <span>Period</span>
                  <span className="font-medium text-foreground">
                    {format(new Date(report.period_start), "MMM d")} - {format(new Date(report.period_end), "MMM d, yyyy")}
                  </span>
                </div>
                {report.summary && (
                  <>
                    <div className="flex justify-between">
                      <span>Deployments</span>
                      <span className="font-medium text-foreground">{report.summary.total_deploys}</span>
                    </div>
                    <div className="flex justify-between">
                      <span>Compliant</span>
                      <span className="font-medium text-emerald-500">{report.summary.approved_pct}%</span>
                    </div>
                  </>
                )}
              </div>

              <button
                disabled={report.status !== "ready"}
                className="flex items-center justify-center gap-2 w-full py-2 bg-secondary text-secondary-foreground rounded-md text-sm font-medium hover:bg-secondary/80 transition-colors disabled:opacity-50"
              >
                <DownloadSimple />
                Download PDF
              </button>
            </div>
          ))}
        </div>
      )}
    </div>
  );
}
