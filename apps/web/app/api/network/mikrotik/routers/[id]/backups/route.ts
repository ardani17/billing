import { NextResponse } from "next/server";
import { networkApi } from "../../../../../../lib/network-api";

export async function GET(request: Request, { params }: { params: Promise<{ id: string }> }) {
  try {
    const { id } = await params;
    const url = new URL(request.url);
    const data = await networkApi(`/api/v1/mikrotik/routers/${id}/backups${url.search}`);
    return NextResponse.json(data);
  } catch (error) {
    return NextResponse.json(
      { success: false, error: { code: "BACKUP_LIST_FAILED", message: error instanceof Error ? error.message : "Gagal mengambil backup router" } },
      { status: 502 },
    );
  }
}

export async function POST(_request: Request, { params }: { params: Promise<{ id: string }> }) {
  try {
    const { id } = await params;
    const data = await networkApi(`/api/v1/mikrotik/routers/${id}/backups`, { method: "POST", body: "{}" });
    return NextResponse.json(data);
  } catch (error) {
    return NextResponse.json(
      { success: false, error: { code: "BACKUP_CREATE_FAILED", message: error instanceof Error ? error.message : "Gagal membuat backup router" } },
      { status: 502 },
    );
  }
}
