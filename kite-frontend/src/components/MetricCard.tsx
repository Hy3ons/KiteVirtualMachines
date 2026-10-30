import type { ReactNode } from 'react';

type MetricCardProps = { readonly label: string; readonly value: ReactNode; readonly hint?: string; readonly icon?: ReactNode };

// MetricCard presents a current resource total; hint describes its scope or unit.
// Workspace and admin pages supply values derived from their existing API data.
export function MetricCard({ label, value, hint, icon }: MetricCardProps) {
  return <article className="metric-card"><div className="metric-card-label"><span>{label}</span><span aria-hidden="true">{icon}</span></div><div className="metric-card-value">{value}</div>{hint && <p className="metric-card-hint">{hint}</p>}</article>;
}
