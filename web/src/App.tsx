import { useEffect, useState } from "react";

type Status = "checking" | "ok" | "down";

export default function App() {
  const [api, setApi] = useState<Status>("checking");

  useEffect(() => {
    fetch("/health")
      .then((r) => setApi(r.ok ? "ok" : "down"))
      .catch(() => setApi("down"));
  }, []);

  return (
    <main style={{ maxWidth: 720, margin: "4rem auto", padding: "0 1rem" }}>
      <h1>hst-bi</h1>
      <p style={{ color: "var(--muted)" }}>
        Foundation only. Dashboards and the SQL console arrive in phase P4.
      </p>
      <p>API: {api}</p>
    </main>
  );
}
