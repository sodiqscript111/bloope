import { useDeploymentLogs } from "../hooks/useDeployments";
import { formatDateTime } from "../utils/date";

type DeploymentLogsProps = {
  deploymentId: string;
};

export function DeploymentLogs({ deploymentId }: DeploymentLogsProps) {
  const logsQuery = useDeploymentLogs(deploymentId);
  const logs = logsQuery.data ?? [];

  return (
    <section
      className="mt-4 rounded-[var(--radius)] border border-[var(--border)] bg-[var(--card)] p-4"
      aria-labelledby="deployment-logs-title"
    >
      <div className="mb-4">
        <div>
          <h3
            id="deployment-logs-title"
            className="text-sm font-semibold uppercase tracking-[0.14em] text-[var(--foreground)]"
          >
            Logs
          </h3>
          <p className="mt-2 text-sm leading-6 text-[var(--muted-foreground)]">
            Streaming live while the worker runs.
          </p>
        </div>
      </div>

      {logsQuery.isLoading ? (
        <p className="rounded-[var(--radius)] border border-dashed border-[var(--border)] bg-[var(--muted)] p-4 text-sm text-[var(--muted-foreground)]">
          Loading logs...
        </p>
      ) : null}
      {logsQuery.isError ? (
        <p className="rounded-[var(--radius)] border border-[var(--danger)]/30 bg-[var(--danger-muted)] p-4 text-sm text-[var(--danger)]">
          Failed to load deployment logs.
        </p>
      ) : null}
      {!logsQuery.isLoading && !logsQuery.isError && logs.length === 0 ? (
        <p className="rounded-[var(--radius)] border border-dashed border-[var(--border)] bg-[var(--muted)] p-4 text-sm leading-6 text-[var(--muted-foreground)]">
          No logs have been recorded for this deployment yet.
        </p>
      ) : null}

      {logs.length > 0 ? (
        <div
          className="grid max-h-80 gap-2 overflow-auto rounded-[var(--radius)] border border-[oklch(0.25_0_0)] bg-[oklch(0.13_0_0)] p-4 font-mono text-xs leading-6 text-[oklch(0.9_0_0)]"
          role="log"
          aria-live="polite"
        >
          {logs.map((logEntry, index) => (
            <div
              className="grid gap-1 sm:grid-cols-[170px_minmax(0,1fr)] sm:gap-4"
              key={`${logEntry.timestamp}-${index}`}
            >
              <time className="whitespace-nowrap text-[oklch(0.65_0_0)]" dateTime={logEntry.timestamp}>
                {formatDateTime(logEntry.timestamp)}
              </time>
              <span className="break-words">{logEntry.message}</span>
            </div>
          ))}
        </div>
      ) : null}
    </section>
  );
}
