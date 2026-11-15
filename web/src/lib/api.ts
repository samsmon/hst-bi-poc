export interface Result {
  columns: string[];
  rows: unknown[][];
  truncated: boolean;
  duration_ms: number;
  sql: string;
}

export interface QueryRequest {
  measures: string[];
  dimensions?: string[];
  grain?: "month" | "quarter" | "year";
  filters?: { dimension: string; in: string[] }[];
  from?: string;
  to?: string;
  order?: { by: string; dir?: "asc" | "desc" }[];
  limit?: number;
}

export interface Failure {
  detail: string;
  rejected?: boolean;
  timeout?: boolean;
}

export interface Column {
  name: string;
  type: string;
  description?: string;
}

export interface Table {
  schema: string;
  name: string;
  kind: string;
  rows: number;
  columns: Column[];
}

async function send<T>(path: string, init?: RequestInit): Promise<T> {
  let res: Response;
  try {
    res = await fetch(path, init);
  } catch {
    throw { detail: "The server did not answer. Check that it is running." } as Failure;
  }
  const body = await res.json().catch(() => ({}));
  if (!res.ok) throw { detail: body.detail ?? `Request failed (${res.status})`, ...body } as Failure;
  return body as T;
}

const post = (body: unknown): RequestInit => ({
  method: "POST",
  headers: { "Content-Type": "application/json" },
  body: JSON.stringify(body),
});

export const query = (req: QueryRequest) => send<Result>("/api/query", post(req));
export const runSql = (text: string) => send<Result>("/api/sql", post({ text }));
export const schema = () => send<{ tables: Table[] }>("/api/schema");
