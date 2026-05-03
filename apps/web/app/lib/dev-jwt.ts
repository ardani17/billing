import { createHmac } from "node:crypto";

function base64Url(input: string) {
  return Buffer.from(input).toString("base64url");
}

export function createDevJwt() {
  const secret = process.env.JWT_SECRET || "change-me-to-a-strong-secret";
  const now = Math.floor(Date.now() / 1000);
  const payload = {
    tenant_id: process.env.DEV_TENANT_ID || "11111111-1111-4111-8111-111111111111",
    user_id: process.env.DEV_USER_ID || "22222222-2222-4222-8222-222222222222",
    role: process.env.DEV_USER_ROLE || "tenant_admin",
    iss: "ispboss-web-dev",
    iat: now,
    exp: now + 60 * 60 * 24,
  };
  const header = { alg: "HS256", typ: "JWT" };
  const encodedHeader = base64Url(JSON.stringify(header));
  const encodedPayload = base64Url(JSON.stringify(payload));
  const signature = createHmac("sha256", secret)
    .update(`${encodedHeader}.${encodedPayload}`)
    .digest("base64url");

  return `${encodedHeader}.${encodedPayload}.${signature}`;
}
