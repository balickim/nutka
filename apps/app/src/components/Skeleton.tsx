// Renders placeholder blocks that hold the layout of a ready state while its data loads.

export function Skeleton({ lines = 3, label = "Ładowanie…" }: { lines?: number; label?: string }) {
  return <div className="skeleton" role="status" aria-label={label}>{Array.from({ length: lines }, (_, index) => <span className="skeleton-line" key={index} />)}</div>;
}
