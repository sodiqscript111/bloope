import { useEffect } from "react";
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";

import {
  createDeployment,
  fetchDeploymentLogs,
  fetchDeployments,
  getDeploymentLogsStreamURL,
} from "../api/deployments";
import type { DeploymentLog } from "../types/deployment";

export const deploymentsQueryKey = ["deployments"] as const;
export const deploymentLogsQueryKey = (deploymentId: string | null) =>
  ["deployment-logs", deploymentId] as const;

export function useDeployments() {
  return useQuery({
    queryKey: deploymentsQueryKey,
    queryFn: fetchDeployments,
    refetchInterval: 2500,
  });
}

export function useDeploymentLogs(deploymentId: string | null) {
  const queryClient = useQueryClient();
  const query = useQuery({
    queryKey: deploymentLogsQueryKey(deploymentId),
    queryFn: () => {
      if (!deploymentId) {
        return Promise.resolve([]);
      }

      return fetchDeploymentLogs(deploymentId);
    },
    enabled: Boolean(deploymentId),
    refetchInterval: false,
    staleTime: Infinity,
  });

  useEffect(() => {
    if (!deploymentId || !query.isSuccess) {
      return;
    }

    const eventSource = new EventSource(getDeploymentLogsStreamURL(deploymentId));

    function handleLogEvent(event: MessageEvent<string>) {
      const logEntry = JSON.parse(event.data) as DeploymentLog;

      queryClient.setQueryData<DeploymentLog[]>(
        deploymentLogsQueryKey(deploymentId),
        (currentLogs = []) => appendUniqueLog(currentLogs, logEntry),
      );
    }

    eventSource.addEventListener("log", handleLogEvent);

    return () => {
      eventSource.removeEventListener("log", handleLogEvent);
      eventSource.close();
    };
  }, [deploymentId, query.isSuccess, queryClient]);

  return query;
}

export function useCreateDeployment() {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: createDeployment,
    onSuccess: () => {
      void queryClient.invalidateQueries({ queryKey: deploymentsQueryKey });
    },
  });
}

function appendUniqueLog(logs: DeploymentLog[], logEntry: DeploymentLog) {
  const nextLogKey = getLogKey(logEntry);
  if (logs.some((existingLog) => getLogKey(existingLog) === nextLogKey)) {
    return logs;
  }

  return [...logs, logEntry];
}

function getLogKey(logEntry: DeploymentLog) {
  return `${logEntry.deployment_id}:${logEntry.timestamp}:${logEntry.message}`;
}
