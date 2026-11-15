const int = new Intl.NumberFormat("en-US", { maximumFractionDigits: 0 });
const dec1 = new Intl.NumberFormat("en-US", { minimumFractionDigits: 1, maximumFractionDigits: 1 });

export const integer = (v: unknown) => (typeof v === "number" ? int.format(v) : "");

/** Rupiah in the largest sensible unit: T (trillion), B (billion), M. */
export function idr(v: unknown): string {
  if (typeof v !== "number") return "";
  const a = Math.abs(v);
  if (a >= 1e12) return `IDR ${(v / 1e12).toFixed(3)} T`;
  if (a >= 1e9) return `IDR ${dec1.format(v / 1e9)} B`;
  if (a >= 1e6) return `IDR ${dec1.format(v / 1e6)} M`;
  return `IDR ${int.format(v)}`;
}

export const percent = (v: unknown, digits = 2) => (typeof v === "number" ? `${(v * 100).toFixed(digits)}%` : "");
export const days = (v: unknown) => (typeof v === "number" ? `${dec1.format(v)} days` : "");

export function byFormat(format: string | undefined, v: unknown): string {
  switch (format) {
    case "integer": return integer(v);
    case "idr": return idr(v);
    case "percent": return percent(v);
    case "days": return days(v);
    default: return cell(v);
  }
}

/** Plain cell text for the SQL results grid. */
export function cell(v: unknown): string {
  if (v === null || v === undefined) return "";
  if (typeof v === "number") return Number.isInteger(v) ? int.format(v) : String(v);
  return String(v);
}

export const isNumber = (v: unknown) => typeof v === "number";
