export type DeploymentStatus = "pending" | "building" | "deploying" | "running" | "failed";

export type Deployment = {
  id: string;
  repo_url: string;
  status: DeploymentStatus;
  image_tag: string;
  live_url: string;
  error_message: string;
  detected_project_type: string;
  detected_framework: string;
  start_command: string;
  env_var_keys: string[];
  readiness_hints: string[];
  container_name: string;
  container_id: string;
  host_port: number;
  source_path: string;
  created_at: string;
  updated_at: string;
};

export type DeploymentLog = {
  deployment_id: string;
  message: string;
  timestamp: string;
};
