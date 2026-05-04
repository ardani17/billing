import { NextResponse } from "next/server";
import { networkApi } from "../../../../../../../lib/network-api";

export async function GET(_request: Request, { params }: { params: Promise<{ id: string }> }) {
  try {
    const { id } = await params;
    const data = await networkApi(`/api/v1/mikrotik/routers/${id}/dhcp/servers`);
    return NextResponse.json(data);
  } catch (error) {
    return NextResponse.json({ success: false, error: { code: "DHCP_SERVERS_FAILED", message: error instanceof Error ? error.message : "Gagal mengambil DHCP server" } }, { status: 502 });
  }
}
