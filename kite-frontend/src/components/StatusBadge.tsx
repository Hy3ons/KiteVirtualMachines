const phaseTones: Readonly<Record<string, string>> = {
  Running: 'success', Ready: 'success', Active: 'success', connected: 'success', Claimed: 'success',
  Creating: 'warning', Provisioning: 'warning', Terminating: 'warning', Deleting: 'warning',
  Pending: 'warning', Reconciling: 'warning', Blocked: 'warning', connecting: 'warning',
  Failed: 'danger', failed: 'danger',
};

// StatusBadge presents phase with text and a redundant dot in workspace surfaces.
// Unknown phases stay visible and neutral instead of implying a successful state.
export function StatusBadge({ phase }: { readonly phase: string }) {
  return <span className={`status-badge status-badge--${phaseTones[phase] ?? 'neutral'}`}><span className="status-dot" aria-hidden="true" />{phase}</span>;
}
