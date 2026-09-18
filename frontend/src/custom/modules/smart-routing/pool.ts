// The API stores a primary separately; the editor exposes one complete pool.
export function routingPool(primary: number | null, candidates: number[]): number[] {
  return [...new Set([...(primary == null ? [] : [primary]), ...candidates])]
}

export function splitRoutingPool(ids: number[]): { primary: number | null; candidates: number[] } {
  const pool = [...new Set(ids)]
  return { primary: pool[0] ?? null, candidates: pool.slice(1) }
}
