import type { Deployment } from "../types/deployment";
import { formatDateTime } from "../utils/date";
import { DeploymentLogs } from "./DeploymentLogs";
import { DeploymentTimeline } from "./DeploymentTimeline";
import { StatusBadge } from "./StatusBadge";

type DeploymentDetailsProps = {
  deployment: Deployment | null;
};

const labelClass =
  "mb-1 text-xs font-semibold uppercase tracking-[0.14em] text-[var(--muted-foreground)]";
const valueClass = "m-0 break-words text-sm leading-6 text-[var(--foreground)]";

export function DeploymentDetails({ deployment }: DeploymentDetailsProps) {
  if (!deployment) {
    return (
      <aside
        className="rounded-[var(--radius)] border border-[var(--border)] bg-[var(--card)] p-5 sm:p-6 lg:sticky lg:top-6"
        aria-labelledby="deployment-details-title"
      >
        <h2
          id="deployment-details-title"
          className="text-base font-semibold uppercase tracking-[0.14em] text-[var(--foreground)]"
        >
          Deployment details
        </h2>
        <p className="mt-5 rounded-[var(--radius)] border border-dashed border-[var(--border)] bg-[var(--muted)] p-4 text-sm leading-6 text-[var(--muted-foreground)]">
          Select a deployment to inspect its full details.
        </p>
      </aside>
    );
  }

  return (
    <aside
      className="rounded-[var(--radius)] border border-[var(--border)] bg-[var(--card)] p-5 sm:p-6 lg:sticky lg:top-6"
      aria-labelledby="deployment-details-title"
    >
      <div className="mb-5 flex items-start justify-between gap-4">
        <div>
          <h2
            id="deployment-details-title"
            className="text-base font-semibold uppercase tracking-[0.14em] text-[var(--foreground)]"
          >
            Deployment details
          </h2>
          <p className="mt-2 font-mono text-sm text-[var(--muted-foreground)]">{deployment.id}</p>
        </div>
        <StatusBadge status={deployment.status} />
      </div>

      <DeploymentTimeline status={deployment.status} />

      <DeploymentLogs deploymentId={deployment.id} />

      <section
        className="mt-4 rounded-[var(--radius)] border border-[var(--border)] bg-[var(--muted)] p-4"
        aria-labelledby="source-insights-title"
      >
        <h3
          id="source-insights-title"
          className="text-sm font-semibold uppercase tracking-[0.14em] text-[var(--foreground)]"
        >
          Source insights
        </h3>
        <dl className="mt-4 grid gap-4 sm:grid-cols-2">
          <div>
            <dt className={labelClass}>Project type</dt>
            <dd className={valueClass}>{deployment.detected_project_type || "Not detected yet"}</dd>
          </div>
          <div>
            <dt className={labelClass}>Framework</dt>
            <dd className={valueClass}>{deployment.detected_framework || "Not detected yet"}</dd>
          </div>
          <div className="sm:col-span-2">
            <dt className={labelClass}>Start command</dt>
            <dd className={`${valueClass} font-mono`}>{deployment.start_command || "Using image default"}</dd>
          </div>
          <div className="sm:col-span-2">
            <dt className={labelClass}>Env vars</dt>
            <dd className={valueClass}>
              {deployment.env_var_keys?.length ? (
                <ul className="grid gap-1 font-mono text-xs">
                  {deployment.env_var_keys.map((key) => (
                    <li key={key}>{key}=******</li>
                  ))}
                </ul>
              ) : (
                "None configured"
              )}
            </dd>
          </div>
        </dl>
        {deployment.readiness_hints?.length > 0 ? (
          <ul className="mt-4 grid gap-2 border-t border-[var(--border)] pt-4 text-sm leading-6 text-[var(--muted-foreground)]">
            {deployment.readiness_hints.map((hint) => (
              <li className="flex gap-2" key={hint}>
                <span aria-hidden="true">-</span>
                <span>{hint}</span>
              </li>
            ))}
          </ul>
        ) : (
          <p className="mt-4 text-sm text-[var(--muted-foreground)]">No readiness hints yet.</p>
        )}
      </section>

      {deployment.status === "failed" ? (
        <div className="mt-4 grid gap-1 rounded-[var(--radius)] border border-[var(--danger)]/30 bg-[var(--danger-muted)] p-4 text-sm text-[var(--danger)]">
          <strong>Deployment failed</strong>
          <span>{deployment.error_message || "The fake worker reported a failed deployment."}</span>
        </div>
      ) : null}

      <dl className="mt-5 grid gap-4">
        <div className="border-b border-[var(--border)] pb-4">
          <dt className={labelClass}>ID</dt>
          <dd className={`${valueClass} font-mono`}>{deployment.id}</dd>
        </div>
        <div className="border-b border-[var(--border)] pb-4">
          <dt className={labelClass}>Repository</dt>
          <dd className={valueClass}>{deployment.repo_url}</dd>
        </div>
        <div className="border-b border-[var(--border)] pb-4">
          <dt className={labelClass}>Status</dt>
          <dd className={valueClass}>{deployment.status}</dd>
        </div>
        <div className="border-b border-[var(--border)] pb-4">
          <dt className={labelClass}>Project type</dt>
          <dd className={valueClass}>{deployment.detected_project_type || "Not detected yet"}</dd>
        </div>
        <div className="border-b border-[var(--border)] pb-4">
          <dt className={labelClass}>Framework</dt>
          <dd className={valueClass}>{deployment.detected_framework || "Not detected yet"}</dd>
        </div>
        <div className="border-b border-[var(--border)] pb-4">
          <dt className={labelClass}>Start command</dt>
          <dd className={`${valueClass} font-mono`}>{deployment.start_command || "Using image default"}</dd>
        </div>
        <div className="border-b border-[var(--border)] pb-4">
          <dt className={labelClass}>Env vars</dt>
          <dd className={valueClass}>
            {deployment.env_var_keys?.length ? deployment.env_var_keys.join(", ") : "None configured"}
          </dd>
        </div>
        <div className="border-b border-[var(--border)] pb-4">
          <dt className={labelClass}>Container</dt>
          <dd className={valueClass}>{deployment.container_name || "Not started yet"}</dd>
        </div>
        <div className="border-b border-[var(--border)] pb-4">
          <dt className={labelClass}>Container ID</dt>
          <dd className={`${valueClass} font-mono`}>{deployment.container_id || "Not started yet"}</dd>
        </div>
        <div className="border-b border-[var(--border)] pb-4">
          <dt className={labelClass}>Host port</dt>
          <dd className={valueClass}>{deployment.host_port || "Not assigned yet"}</dd>
        </div>
        <div className="border-b border-[var(--border)] pb-4">
          <dt className={labelClass}>Source path</dt>
          <dd className={valueClass}>{deployment.source_path || "Not cloned yet"}</dd>
        </div>
        <div className="border-b border-[var(--border)] pb-4">
          <dt className={labelClass}>Image tag</dt>
          <dd className={valueClass}>{deployment.image_tag || "Not assigned yet"}</dd>
        </div>
        <div className="border-b border-[var(--border)] pb-4">
          <dt className={labelClass}>Live URL</dt>
          <dd className={valueClass}>
            {deployment.live_url ? (
              <a
                className="border-b border-[var(--border)] transition hover:border-[var(--foreground)] hover:text-[var(--muted-foreground)]"
                href={deployment.live_url}
                target="_blank"
                rel="noreferrer"
              >
                {deployment.live_url}
              </a>
            ) : (
              "Not assigned yet"
            )}
          </dd>
        </div>
        <div className="border-b border-[var(--border)] pb-4">
          <dt className={labelClass}>Error message</dt>
          <dd className={valueClass}>{deployment.error_message || "None"}</dd>
        </div>
        <div className="border-b border-[var(--border)] pb-4">
          <dt className={labelClass}>Created at</dt>
          <dd className={valueClass}>{formatDateTime(deployment.created_at)}</dd>
        </div>
        <div>
          <dt className={labelClass}>Updated at</dt>
          <dd className={valueClass}>{formatDateTime(deployment.updated_at)}</dd>
        </div>
      </dl>
    </aside>
  );
}
