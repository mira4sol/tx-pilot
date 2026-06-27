const CLUSTER_ALIASES: Record<string, string> = {
  "mainnet-beta": "mainnet",
  mainnet: "mainnet",
  devnet: "devnet",
  testnet: "testnet",
};

function normalizeCluster(cluster?: string): string {
  if (!cluster) return "mainnet";
  return CLUSTER_ALIASES[cluster] ?? cluster;
}

export function solscanTxUrl(signature: string, cluster?: string): string {
  const c = normalizeCluster(cluster);
  const base = `https://solscan.io/tx/${signature}`;
  return c === "mainnet" ? base : `${base}?cluster=${c}`;
}

export function explorerSlotUrl(slot: number, cluster?: string): string {
  if (slot <= 0) return "";
  const c = normalizeCluster(cluster);
  const suffix = c === "mainnet" ? "" : `?cluster=${c}`;
  return `https://explorer.solana.com/block/${slot}${suffix}`;
}

export function explorerAddressUrl(address: string, cluster?: string): string {
  const c = normalizeCluster(cluster);
  const suffix = c === "mainnet" ? "" : `?cluster=${c}`;
  return `https://explorer.solana.com/address/${address}${suffix}`;
}
