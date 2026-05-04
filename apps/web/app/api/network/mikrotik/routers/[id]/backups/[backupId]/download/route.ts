import { createDevJwt } from "../../../../../../../../lib/dev-jwt";

const networkApiBaseUrl = process.env.NETWORK_API_URL || "http://localhost:3002";

export async function GET(_request: Request, { params }: { params: Promise<{ id: string; backupId: string }> }) {
  const { id, backupId } = await params;
  const response = await fetch(`${networkApiBaseUrl}/api/v1/mikrotik/routers/${id}/backups/${backupId}/download`, {
    headers: { Authorization: `Bearer ${createDevJwt()}` },
    cache: "no-store",
  });
  const body = await response.text();
  return new Response(body, {
    status: response.status,
    headers: {
      "Content-Type": response.headers.get("content-type") || "text/plain; charset=utf-8",
      "Content-Disposition": response.headers.get("content-disposition") || "attachment",
    },
  });
}
