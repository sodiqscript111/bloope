import type { Deployment, DeploymentLog } from "../types/deployment";

const API_BASE_URL = (import.meta.env.VITE_API_BASE_URL ?? "/api").replace(/\/$/, "");

type CreateDeploymentInput = {
  repo_url: string;
  env_vars?: Record<string, string>;
};

async function request<T>(path: string, options?: RequestInit): Promise<T> {
  const response = await fetch(`${API_BASE_URL}${path}`, {
    headers: {
      "Content-Type": "application/json",
      ...options?.headers,
    },
    ...options,
  });

  if (!response.ok) {
    const message = await readErrorMessage(response);
    throw new Error(message);
  }

  return response.json() as Promise<T>;
}

async function readErrorMessage(response: Response) {
  try {
    const body = (await response.json()) as { error?: string };
    return body.error ?? `Request failed with status ${response.status}`;
  } catch {
    return `Request failed with status ${response.status}`;
  }
}

export function fetchDeployments() {
  return request<Deployment[]>("/deployments");
}

export function fetchDeploymentLogs(deploymentId: string) {
  return request<DeploymentLog[]>(`/deployments/${deploymentId}/logs`);
}

export function getDeploymentLogsStreamURL(deploymentId: string) {
  return `${API_BASE_URL}/deployments/${deploymentId}/logs/stream`;
}

export function createDeployment(input: CreateDeploymentInput) {
  return request<Deployment>("/deployments", {
    method: "POST",
    body: JSON.stringify(input),
  });
}
