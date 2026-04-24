import { useEffect, useState } from "react";

import { DeploymentDetails } from "./components/DeploymentDetails";
import { DeploymentForm } from "./components/DeploymentForm";
import { DeploymentList } from "./components/DeploymentList";
import { useDeployments } from "./hooks/useDeployments";

function App() {
  const [selectedDeploymentId, setSelectedDeploymentId] = useState<string | null>(null);
  const deploymentsQuery = useDeployments();
  const deployments = deploymentsQuery.data ?? [];
  const selectedDeployment =
    deployments.find((deployment) => deployment.id === selectedDeploymentId) ?? null;

  useEffect(() => {
    if (!selectedDeploymentId && deployments.length > 0) {
      setSelectedDeploymentId(deployments[0].id);
      return;
    }

    if (selectedDeploymentId && deployments.length > 0 && !selectedDeployment) {
      setSelectedDeploymentId(deployments[0].id);
    }
  }, [deployments, selectedDeployment, selectedDeploymentId]);

  return (
    <main className="min-h-screen bg-[var(--background)] text-[var(--foreground)]">
      <div className="mx-auto w-full max-w-7xl px-5 py-14 sm:px-8 sm:py-20">
        <header className="mb-10 grid gap-8 border-b border-[var(--border)] pb-10 lg:grid-cols-[minmax(0,1fr)_320px] lg:items-end">
          <div>
            <p className="mb-4 text-sm font-semibold uppercase tracking-[0.18em] text-[var(--muted-foreground)]">
              Bloope MVP
            </p>
            <h1 className="max-w-4xl text-5xl font-semibold leading-none tracking-normal text-[var(--foreground)] sm:text-7xl">
              Deploy software from a GitHub URL.
            </h1>
          </div>
          <p className="max-w-xl text-base leading-8 text-[var(--muted-foreground)]">
            Clone, detect, build with Railpack, run in Docker, and route through Caddy. A small
            local deployment desk with live logs and no ceremony.
          </p>
        </header>

        <DeploymentForm />

        <section
          className="grid gap-6 lg:grid-cols-[minmax(0,0.95fr)_minmax(420px,1.05fr)] lg:items-start"
          aria-label="Deployments dashboard"
        >
          <DeploymentList
            deployments={deployments}
            isLoading={deploymentsQuery.isLoading}
            isError={deploymentsQuery.isError}
            selectedDeploymentId={selectedDeploymentId}
            onSelectDeployment={setSelectedDeploymentId}
          />
          <DeploymentDetails deployment={selectedDeployment} />
        </section>
      </div>
    </main>
  );
}

export default App;
