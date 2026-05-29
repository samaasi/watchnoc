import type { DeployEvent, Approval, EvidenceReport, DORASnapshot } from "./types";

const BASE_URL = process.env.NEXT_PUBLIC_API_URL ?? "http://localhost:8080/api/v1";

async function apiFetch<T>(path: string, token: string | null, init?: RequestInit): Promise<T> {
  const headers: HeadersInit = { "Content-Type": "application/json" };
  if (token) {
    headers["Authorization"] = `Bearer ${token}`;
  }

  const res = await fetch(`${BASE_URL}${path}`, {
    ...init,
    headers: { ...headers, ...init?.headers },
  });
  
  if (!res.ok) {
    const body = await res.json().catch(() => ({}));
    throw new Error(body?.message ?? `API error ${res.status}`);
  }
  return res.json() as Promise<T>;
}

export const api = {
  deploys: {
    list: (orgID: string, token: string | null) =>
      apiFetch<{ data: DeployEvent[] }>(`/orgs/${orgID}/deploys`, token),
    get: (orgID: string, id: string, token: string | null) =>
      apiFetch<DeployEvent>(`/orgs/${orgID}/deploys/${id}`, token),
  },
  approvals: {
    approve: (orgID: string, deployID: string, token: string | null, comment?: string) =>
      apiFetch<Approval>(`/orgs/${orgID}/deploys/${deployID}/approve`, token, {
        method: "POST",
        body: JSON.stringify({ comment }),
      }),
    reject: (orgID: string, deployID: string, token: string | null, comment?: string) =>
      apiFetch<Approval>(`/orgs/${orgID}/deploys/${deployID}/reject`, token, {
        method: "POST",
        body: JSON.stringify({ comment }),
      }),
  },
  evidence: {
    list: (orgID: string, token: string | null) =>
      apiFetch<{ data: EvidenceReport[] }>(`/orgs/${orgID}/evidence`, token),
    generate: (orgID: string, framework: string, periodStart: string, periodEnd: string, token: string | null) =>
      apiFetch<EvidenceReport>(`/orgs/${orgID}/evidence`, token, {
        method: "POST",
        body: JSON.stringify({ framework, period_start: periodStart, period_end: periodEnd }),
      }),
  },
  dora: {
    latest: (orgID: string, token: string | null) =>
      apiFetch<DORASnapshot>(`/orgs/${orgID}/dora/latest`, token),
  },
};
