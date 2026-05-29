"use client";

import { useQuery } from "@tanstack/react-query";
import { useApi } from "@/lib/useApi";
import { ChartLineUp, Clock, Lightning, Warning } from "@phosphor-icons/react";

function MetricCard({ 
  title, 
  value, 
  tier, 
  icon: Icon,
  trend
}: { 
  title: string; 
  value: string; 
  tier: string;
  icon: React.ElementType;
  trend?: string;
}) {
  const isElite = tier.toLowerCase() === "elite";
  const isHigh = tier.toLowerCase() === "high";
  const isMedium = tier.toLowerCase() === "medium";
  
  return (
    <div className="rounded-xl border border-border bg-card shadow-sm p-6 flex flex-col">
      <div className="flex items-center justify-between mb-4">
        <h3 className="font-medium text-sm text-muted-foreground">{title}</h3>
        <div className="p-2 bg-muted rounded-md text-foreground">
          <Icon className="size-4" />
        </div>
      </div>
      <div className="text-3xl font-bold mb-2">{value}</div>
      <div className="flex items-center gap-3">
        <span className={`inline-flex items-center rounded-full px-2 py-0.5 text-xs font-medium uppercase tracking-wider ${
          isElite ? "bg-emerald-500/10 text-emerald-500" :
          isHigh ? "bg-blue-500/10 text-blue-500" :
          isMedium ? "bg-amber-500/10 text-amber-500" :
          "bg-red-500/10 text-red-500"
        }`}>
          {tier}
        </span>
        {trend && <span className="text-xs text-muted-foreground">{trend}</span>}
      </div>
    </div>
  );
}

export default function DORAPage() {
  const { getBoundApi, hasOrg } = useApi();

  const { data: snapshot, isLoading, error } = useQuery({
    queryKey: ["dora", "latest"],
    queryFn: async () => {
      const boundApi = await getBoundApi();
      return await boundApi.dora.latest();
    },
    enabled: hasOrg,
  });

  if (!hasOrg) return <div className="p-8">Please select an organization.</div>;
  if (isLoading) return <div className="p-8">Loading metrics...</div>;
  if (error) return <div className="p-8 text-red-500">Error loading metrics.</div>;
  if (!snapshot) return <div className="p-8">No DORA metrics found.</div>;

  return (
    <div className="flex flex-col gap-8 p-8 max-w-5xl">
      <div>
        <h1 className="text-2xl font-bold tracking-tight">DORA Metrics</h1>
        <p className="text-muted-foreground mt-1">
          Measure delivery performance and operational stability.
        </p>
      </div>

      <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-4 gap-4">
        <MetricCard
          title="Deployment Frequency"
          value={`${snapshot.deployment_frequency}/day`}
          tier={snapshot.deployment_frequency_tier}
          icon={Lightning}
          trend="+12% from last month"
        />
        <MetricCard
          title="Lead Time for Changes"
          value={`${snapshot.lead_time_hours}h`}
          tier={snapshot.lead_time_tier}
          icon={Clock}
          trend="-2.1h from last month"
        />
        <MetricCard
          title="Mean Time to Restore"
          value={`${snapshot.mttr_hours}h`}
          tier={snapshot.mttr_tier}
          icon={ChartLineUp}
          trend="No change"
        />
        <MetricCard
          title="Change Failure Rate"
          value={`${snapshot.change_failure_rate}%`}
          tier={snapshot.change_failure_tier}
          icon={Warning}
          trend="-0.5% from last month"
        />
      </div>
      
      <div className="rounded-xl border border-border bg-card shadow-sm p-6 mt-4">
         <h2 className="text-lg font-semibold mb-4">Summary (Last 30 Days)</h2>
         <div className="flex gap-12">
            <div className="flex flex-col gap-1">
               <span className="text-sm text-muted-foreground">Total Deployments</span>
               <span className="text-2xl font-bold">{snapshot.total_deploys}</span>
            </div>
            <div className="flex flex-col gap-1">
               <span className="text-sm text-muted-foreground">Production Incidents</span>
               <span className="text-2xl font-bold">{snapshot.total_incidents}</span>
            </div>
         </div>
      </div>
    </div>
  );
}
