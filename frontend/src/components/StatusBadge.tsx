import type { DeploymentStatus } from "../types/deployment";

type StatusBadgeProps = {
  status: DeploymentStatus;
};

const statusClasses: Record<DeploymentStatus, string> = {
  pending: "border-[var(--border)] bg-[var(--muted)] text-[var(--muted-foreground)]",
  building: "border-[var(--warning)]/30 bg-[var(--warning-muted)] text-[var(--warning)]",
  deploying: "border-[var(--warning)]/30 bg-[var(--warning-muted)] text-[var(--warning)]",
  running: "border-[var(--success)]/30 bg-[var(--success-muted)] text-[var(--success)]",
  failed: "border-[var(--danger)]/30 bg-[var(--danger-muted)] text-[var(--danger)]",
};

export function StatusBadge({ status }: StatusBadgeProps) {
  return (
    <span
      className={`inline-flex items-center rounded-full border px-2.5 py-1 text-xs font-semibold uppercase tracking-[0.12em] ${statusClasses[status]}`}
    >
      {status}
    </span>
  );
}
