// Accepts a decimal TCP port entered in gateway settings or returned for SSH instructions.
export function validTCPPort(value: string): boolean {
  const normalized = value.trim();
  const port = Number(normalized);
  return /^\d+$/.test(normalized) && Number.isInteger(port) && port >= 1 && port <= 65535;
}
