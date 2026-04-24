import { type Dispatch, type FormEvent, type SetStateAction, useState } from "react";

import { useCreateDeployment } from "../hooks/useDeployments";

const envVarNamePattern = /^[A-Za-z_][A-Za-z0-9_]*$/;
const envValuePlaceholder = "https://example.com, redis://..., token, secret, or any runtime value";

type EnvVarDraft = {
  id: string;
  key: string;
  value: string;
};

export function DeploymentForm() {
  const [repoURL, setRepoURL] = useState("");
  const [envRows, setEnvRows] = useState<EnvVarDraft[]>([createEnvVarDraft()]);
  const [envError, setEnvError] = useState("");
  const createDeployment = useCreateDeployment();

  function handleSubmit(event: FormEvent<HTMLFormElement>) {
    event.preventDefault();

    const trimmedRepoURL = repoURL.trim();
    if (!trimmedRepoURL || createDeployment.isPending) {
      return;
    }

    let envVars: Record<string, string>;
    try {
      envVars = parseEnvVars(envRows);
      setEnvError("");
    } catch (error) {
      setEnvError(error instanceof Error ? error.message : "Invalid environment variables.");
      return;
    }

    createDeployment.mutate(
      { repo_url: trimmedRepoURL, env_vars: envVars },
      {
        onSuccess: () => {
          setRepoURL("");
          setEnvRows([createEnvVarDraft()]);
          setEnvError("");
        },
      },
    );
  }

  return (
    <section
      className="mb-8 grid gap-6 rounded-[var(--radius)] border border-[var(--border)] bg-[var(--card)] p-5 sm:p-6 lg:grid-cols-[260px_minmax(0,1fr)]"
      aria-labelledby="deployment-form-title"
    >
      <div>
        <h2
          id="deployment-form-title"
          className="text-base font-semibold uppercase tracking-[0.14em] text-[var(--foreground)]"
        >
          New deployment
        </h2>
        <p className="mt-3 text-sm leading-7 text-[var(--muted-foreground)]">
          Paste a GitHub repository URL and optional runtime environment variables.
        </p>
      </div>

      <form className="grid gap-3" onSubmit={handleSubmit}>
        <label
          className="text-xs font-semibold uppercase tracking-[0.14em] text-[var(--muted-foreground)]"
          htmlFor="repo-url"
        >
          GitHub repository URL
        </label>
        <div className="flex flex-col gap-3 sm:flex-row">
          <input
            id="repo-url"
            type="url"
            value={repoURL}
            onChange={(event) => setRepoURL(event.target.value)}
            placeholder="https://github.com/example/app"
            disabled={createDeployment.isPending}
            className="min-h-12 w-full rounded-[var(--radius)] border border-[var(--border)] bg-[var(--background)] px-4 text-sm text-[var(--foreground)] outline-none transition focus:border-[var(--foreground)] disabled:opacity-60"
            required
          />
          <button
            className="inline-flex min-h-12 items-center justify-center gap-2 rounded-[var(--radius)] border border-[var(--foreground)] bg-[var(--foreground)] px-6 text-sm font-semibold text-[var(--background)] transition hover:border-white hover:bg-white disabled:opacity-50"
            type="submit"
            disabled={!repoURL.trim() || createDeployment.isPending}
          >
            {createDeployment.isPending ? (
              <>
                <span
                  className="h-3.5 w-3.5 rounded-full border-2 border-[var(--background)]/40 border-t-[var(--background)] [animation:spin_0.8s_linear_infinite]"
                  aria-hidden="true"
                />
                Deploying
              </>
            ) : (
              "Deploy"
            )}
          </button>
        </div>

        <label
          className="mt-3 text-xs font-semibold uppercase tracking-[0.14em] text-[var(--muted-foreground)]"
          htmlFor="deployment-env-name-0"
        >
          Environment variables <span className="font-normal normal-case tracking-normal">(optional)</span>
        </label>
        <div className="rounded-[var(--radius)] border border-[var(--border)] bg-[var(--muted)] p-4">
          <div className="mb-4 flex flex-col gap-2 sm:flex-row sm:items-end sm:justify-between">
            <div>
              <p className="text-sm font-medium text-[var(--foreground)]">Runtime configuration</p>
              <p className="text-sm leading-6 text-[var(--muted-foreground)]">
                Add a variable name on the left and the value, URL, token, or connection string on
                the right.
              </p>
            </div>
            <button
              type="button"
              onClick={() => setEnvRows((currentRows) => [...currentRows, createEnvVarDraft()])}
              disabled={createDeployment.isPending}
              className="inline-flex min-h-10 items-center justify-center rounded-[var(--radius)] border border-[var(--border)] bg-[var(--card)] px-4 text-sm font-medium text-[var(--foreground)] transition hover:border-[var(--foreground)] disabled:opacity-50"
            >
              Add variable
            </button>
          </div>

          <div className="grid gap-3">
            {envRows.map((row, index) => (
              <div
                key={row.id}
                className="grid gap-3 rounded-[var(--radius)] border border-[var(--border)] bg-[var(--card)] p-3 lg:grid-cols-[minmax(0,0.7fr)_minmax(0,1.3fr)_auto]"
              >
                <div className="grid gap-2">
                  <label
                    className="text-[11px] font-semibold uppercase tracking-[0.14em] text-[var(--muted-foreground)]"
                    htmlFor={`deployment-env-name-${index}`}
                  >
                    Name
                  </label>
                  <input
                    id={`deployment-env-name-${index}`}
                    type="text"
                    value={row.key}
                    onChange={(event) =>
                      setEnvRows((currentRows) =>
                        currentRows.map((currentRow, currentIndex) =>
                          currentIndex === index
                            ? { ...currentRow, key: event.target.value }
                            : currentRow,
                        ),
                      )
                    }
                    placeholder="DATABASE_URL"
                    disabled={createDeployment.isPending}
                    className="min-h-11 w-full rounded-[var(--radius)] border border-[var(--border)] bg-[var(--background)] px-3 text-sm text-[var(--foreground)] outline-none transition focus:border-[var(--foreground)] disabled:opacity-60"
                    spellCheck={false}
                  />
                </div>

                <div className="grid gap-2">
                  <label
                    className="text-[11px] font-semibold uppercase tracking-[0.14em] text-[var(--muted-foreground)]"
                    htmlFor={`deployment-env-value-${index}`}
                  >
                    Value
                  </label>
                  <input
                    id={`deployment-env-value-${index}`}
                    type="text"
                    value={row.value}
                    onChange={(event) =>
                      setEnvRows((currentRows) =>
                        currentRows.map((currentRow, currentIndex) =>
                          currentIndex === index
                            ? { ...currentRow, value: event.target.value }
                            : currentRow,
                        ),
                      )
                    }
                    placeholder={envValuePlaceholder}
                    disabled={createDeployment.isPending}
                    className="min-h-11 w-full rounded-[var(--radius)] border border-[var(--border)] bg-[var(--background)] px-3 text-sm text-[var(--foreground)] outline-none transition placeholder:text-[var(--muted-foreground)] focus:border-[var(--foreground)] disabled:opacity-60"
                    spellCheck={false}
                  />
                </div>

                <div className="flex items-end">
                  <button
                    type="button"
                    onClick={() => removeEnvRow(index, setEnvRows)}
                    disabled={createDeployment.isPending}
                    className="inline-flex min-h-11 w-full items-center justify-center rounded-[var(--radius)] border border-[var(--border)] px-4 text-sm font-medium text-[var(--muted-foreground)] transition hover:border-[var(--foreground)] hover:text-[var(--foreground)] disabled:opacity-50 lg:w-auto"
                  >
                    Remove
                  </button>
                </div>
              </div>
            ))}
          </div>

          <p className="mt-4 m-0 text-sm leading-6 text-[var(--muted-foreground)]">
            Variable values are sent to the container at runtime and only the variable names are
            shown later in the UI.
          </p>
        </div>
      </form>

      {envError ? (
        <p
          className="m-0 rounded-[var(--radius)] border border-[var(--danger)]/30 bg-[var(--danger-muted)] p-3 text-sm text-[var(--danger)] lg:col-start-2"
          role="alert"
        >
          {envError}
        </p>
      ) : null}

      {createDeployment.isError ? (
        <p
          className="m-0 rounded-[var(--radius)] border border-[var(--danger)]/30 bg-[var(--danger-muted)] p-3 text-sm text-[var(--danger)] lg:col-start-2"
          role="alert"
        >
          {createDeployment.error.message}
        </p>
      ) : null}
    </section>
  );
}

function parseEnvVars(rows: EnvVarDraft[]) {
  const envVars: Record<string, string> = {};

  rows.forEach((row, index) => {
    const key = row.key.trim();
    const value = row.value;
    if (!key && !value.trim()) {
      return;
    }

    if (!key) {
      throw new Error(`Environment variable row ${index + 1} is missing a name.`);
    }

    if (!envVarNamePattern.test(key)) {
      throw new Error(`Environment variable "${key}" has an invalid name.`);
    }

    envVars[key] = stripMatchingQuotes(value.trim());
  });

  return envVars;
}

function createEnvVarDraft(): EnvVarDraft {
  return {
    id: crypto.randomUUID(),
    key: "",
    value: "",
  };
}

function removeEnvRow(
  index: number,
  setEnvRows: Dispatch<SetStateAction<EnvVarDraft[]>>,
) {
  setEnvRows((currentRows) => {
    if (currentRows.length === 1) {
      return [createEnvVarDraft()];
    }

    return currentRows.filter((_, currentIndex) => currentIndex !== index);
  });
}

function stripMatchingQuotes(value: string) {
  if (
    (value.startsWith('"') && value.endsWith('"')) ||
    (value.startsWith("'") && value.endsWith("'"))
  ) {
    return value.slice(1, -1);
  }

  return value;
}
