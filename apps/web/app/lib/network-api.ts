import { createDevJwt } from "./dev-jwt";

const networkApiBaseUrl = process.env.NETWORK_API_URL || "http://localhost:3002";

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
    const message =
      typeof body === "object" && body && "error" in body
        ? JSON.stringify(body.error)
        : `network-service error ${response.status}`;
    throw new Error(message);
  }

  return body;
}
