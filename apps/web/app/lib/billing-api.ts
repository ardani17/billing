import { createDevJwt } from "./dev-jwt";

const billingApiBaseUrl = process.env.BILLING_API_URL || "http://localhost:3001";

export async function billingApi(path: string, init?: RequestInit) {
  const response = await fetch(`${billingApiBaseUrl}${path}`, {
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
        : `billing-api error ${response.status}`;
    throw new Error(message);
  }

  return body;
}
