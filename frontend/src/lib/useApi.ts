import { useAuth, useOrganization } from "@clerk/nextjs";
import { useCallback } from "react";
import { api } from "./api";

export function useApi() {
  const { getToken } = useAuth();
  const { organization } = useOrganization();
  const orgId = organization?.id;

  const getBoundApi = useCallback(async () => {
    const token = await getToken();
    
    if (!orgId) {
      throw new Error("No organization selected");
    }

    return {
      orgId,
      deploys: {
        list: () => api.deploys.list(orgId, token),
        get: (id: string) => api.deploys.get(orgId, id, token),
      },
      approvals: {
        approve: (deployID: string, comment?: string) => api.approvals.approve(orgId, deployID, token, comment),
        reject: (deployID: string, comment?: string) => api.approvals.reject(orgId, deployID, token, comment),
      },
      evidence: {
        list: () => api.evidence.list(orgId, token),
        generate: (framework: string, periodStart: string, periodEnd: string) => 
          api.evidence.generate(orgId, framework, periodStart, periodEnd, token),
      },
      dora: {
        latest: () => api.dora.latest(orgId, token),
      }
    };
  }, [getToken, orgId]);

  return { getBoundApi, hasOrg: !!orgId };
}
