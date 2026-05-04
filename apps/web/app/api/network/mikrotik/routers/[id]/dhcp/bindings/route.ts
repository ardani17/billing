import { NextResponse } from "next/server";
import { networkApi } from "../../../../../../../lib/network-api";

export async function GET(request: Request, { params }: { params: Promise<{ id: string }> }) {
  try {
    const { id } = await params;
    const url = new URL(request.url);
    const query = url.search || "?page_size=50";
    const data = await networkApi(`/api/v1/mikrotik/routers/${id}/dhcp/bindings${query}`);
    return NextResponse.json(data);
  } catch (error) {
    return NextResponse.json({ success: false, error: { code: "DHCP_BINDINGS_FAILED", message: error instanceof Error ? error.message : "Gagal mengambil DHCP binding" } }, { status: 502 });
  }
}

export async function POST(request: Request, { params }: { params: Promise<{ id: string }> }) {
  try {
    const { id } = await params;
    const body = await request.json();
    const data = await networkApi(`/api/v1/mikrotik/routers/${id}/dhcp/bindings`, {
      method: "POST",
      body: JSON.stringify(body),
    });
    return NextResponse.json(data);
  } catch (error) {
    return NextResponse.json({ success: false, error: { code: "DHCP_BINDING_CREATE_FAILED", message: error instanceof Error ? error.message : "Gagal membuat DHCP binding" } }, { status: 502 });
  }
}
