import { createDevJwt } from "./dev-jwt";

const networkApiBaseUrl = process.env.NETWORK_API_URL || "http://localhost:3002";

export class NetworkApiError extends Error {
  status: number;
  body: unknown;

  constructor(status: number, body: unknown) {
    const message =
      typeof body === "object" && body && "error" in body
        ? JSON.stringify((body as { error: unknown }).error)
        : `network-service error ${status}`;
    super(message);
    this.name = "NetworkApiError";
    this.status = status;
    this.body = body;
  }
}

export async function networkApi(path: string, init?: RequestInit) {
  const response = await fetch(`${networkApiBaseUrl}${path}`, {
    ...init,
    headers: {
      "Content-Type": "application/json",
      Authorization: `Bearer ${createDevJwt()}`,
      ...(init?.headers || {}),
    },
    cache: "no-store",
  });

  const contentType = response.headers.get("content-type") || "";
  const body = contentType.includes("application/json") ? await response.json() : await response.text();

  if (!response.ok) {
    throw new NetworkApiError(response.status, body);
  }

  return body;
}
