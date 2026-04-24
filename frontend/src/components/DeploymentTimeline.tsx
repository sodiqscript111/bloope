import type { DeploymentStatus } from "../types/deployment";

const steps: DeploymentStatus[] = ["pending", "building", "deploying", "running"];

type DeploymentTimelineProps = {
  status: DeploymentStatus;
};

export function DeploymentTimeline({ status }: DeploymentTimelineProps) {
  const currentStepIndex = steps.indexOf(status);
  const isFailed = status === "failed";

  return (
    <section
      className="rounded-[var(--radius)] border border-[var(--border)] bg-[var(--muted)] p-4"
      aria-label="Deployment progress"
    >
      <div className="mb-4 flex items-center justify-between gap-3">
        <h3 className="text-sm font-semibold uppercase tracking-[0.14em] text-[var(--foreground)]">
          Progress
        </h3>
        {isFailed ? (
          <span className="text-xs font-semibold uppercase tracking-[0.14em] text-[var(--danger)]">
            Failed
          </span>
        ) : null}
      </div>

      <ol className="grid gap-3">
        {steps.map((step, index) => {
          const isComplete = !isFailed && currentStepIndex > index;
          const isCurrent = !isFailed && currentStepIndex === index;
          const dotClass = isComplete
            ? "border-[var(--success)] bg-[var(--success)]"
            : isCurrent
              ? "border-[var(--foreground)] bg-[var(--card)] ring-4 ring-[var(--accent)]"
              : "border-[var(--border)] bg-[var(--card)]";

          return (
            <li
              className={`flex items-center gap-3 text-sm font-medium ${
                isComplete || isCurrent ? "text-[var(--foreground)]" : "text-[var(--muted-foreground)]"
              }`}
              key={step}
            >
              <span className={`h-3 w-3 rounded-full border-2 ${dotClass}`} aria-hidden="true" />
              <span>{step}</span>
            </li>
          );
        })}
      </ol>
    </section>
  );
}
