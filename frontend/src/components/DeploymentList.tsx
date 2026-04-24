import type { Deployment } from "../types/deployment";
import { formatDateTime } from "../utils/date";
import { StatusBadge } from "./StatusBadge";

type DeploymentListProps = {
  deployments: Deployment[];
  isLoading: boolean;
  isError: boolean;
  selectedDeploymentId: string | null;
  onSelectDeployment: (id: string) => void;
};

export function DeploymentList({
  deployments,
  isLoading,
  isError,
  selectedDeploymentId,
  onSelectDeployment,
}: DeploymentListProps) {
  return (
    <section
      className="rounded-[var(--radius)] border border-[var(--border)] bg-[var(--card)] p-5 sm:p-6"
      aria-labelledby="deployment-list-title"
    >
      <div className="mb-6 flex items-start justify-between gap-4">
        <div>
          <h2
            id="deployment-list-title"
            className="text-base font-semibold uppercase tracking-[0.14em] text-[var(--foreground)]"
          >
            Deployment runs
          </h2>
          <p className="mt-2 text-sm leading-6 text-[var(--muted-foreground)]">
            Polling every few seconds for status updates.
          </p>
        </div>
        <span className="inline-flex min-w-9 items-center justify-center rounded-full border border-[var(--border)] bg-[var(--muted)] px-3 py-1 text-sm font-semibold text-[var(--muted-foreground)]">
          {deployments.length}
        </span>
      </div>

      {isLoading ? <p className="rounded-[var(--radius)] border border-dashed border-[var(--border)] bg-[var(--muted)] p-4 text-sm text-[var(--muted-foreground)]">Loading deployments...</p> : null}
      {isError ? (
        <p className="rounded-[var(--radius)] border border-[var(--danger)]/30 bg-[var(--danger-muted)] p-4 text-sm text-[var(--danger)]">
          Failed to load deployments.
        </p>
      ) : null}
      {!isLoading && !isError && deployments.length === 0 ? (
        <p className="rounded-[var(--radius)] border border-dashed border-[var(--border)] bg-[var(--muted)] p-4 text-sm leading-6 text-[var(--muted-foreground)]">
          No deployments yet. Submit a repository URL to start one.
        </p>
      ) : null}

      <div className="grid gap-3">
        {deployments.map((deployment) => (
          <button
            className={`w-full rounded-[var(--radius)] border bg-[var(--card)] p-4 text-left transition hover:-translate-y-0.5 hover:border-[var(--foreground)] ${
              deployment.id === selectedDeploymentId
                ? "border-[var(--foreground)] shadow-[inset_0_0_0_1px_var(--foreground)]"
                : "border-[var(--border)]"
            }`}
            key={deployment.id}
            type="button"
            onClick={() => onSelectDeployment(deployment.id)}
          >
            <div className="mb-3 flex items-start justify-between gap-3">
              <span className="font-mono text-sm font-semibold text-[var(--foreground)]">
                {deployment.id}
              </span>
              <StatusBadge status={deployment.status} />
            </div>
            <p className="mb-4 break-words text-sm leading-6 text-[var(--muted-foreground)]">
              {deployment.repo_url}
            </p>
            <dl className="grid gap-3 text-sm sm:grid-cols-2">
              <div>
                <dt className="mb-1 text-xs font-semibold uppercase tracking-[0.14em] text-[var(--muted-foreground)]">
                  Image
                </dt>
                <dd className="m-0 break-words text-[var(--foreground)]">{deployment.image_tag || "Pending"}</dd>
              </div>
              <div>
                <dt className="mb-1 text-xs font-semibold uppercase tracking-[0.14em] text-[var(--muted-foreground)]">
                  Live URL
                </dt>
                <dd className="m-0 break-words text-[var(--foreground)]">{deployment.live_url || "Pending"}</dd>
              </div>
              <div>
                <dt className="mb-1 text-xs font-semibold uppercase tracking-[0.14em] text-[var(--muted-foreground)]">
                  Created
                </dt>
                <dd className="m-0 text-[var(--foreground)]">{formatDateTime(deployment.created_at)}</dd>
              </div>
              <div>
                <dt className="mb-1 text-xs font-semibold uppercase tracking-[0.14em] text-[var(--muted-foreground)]">
                  Updated
                </dt>
                <dd className="m-0 text-[var(--foreground)]">{formatDateTime(deployment.updated_at)}</dd>
              </div>
            </dl>
          </button>
        ))}
      </div>
    </section>
  );
}
