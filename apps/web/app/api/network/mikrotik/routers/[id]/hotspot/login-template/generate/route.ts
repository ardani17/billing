import { NextResponse } from "next/server";
import { networkApi } from "../../../../../../../../lib/network-api";

export async function POST(request: Request, { params }: { params: Promise<{ id: string }> }) {
  try {
    const { id } = await params;
    const body = await request.json();
    const data = await networkApi(`/api/v1/mikrotik/routers/${id}/hotspot/login-template/generate`, {
      method: "POST",
      body: JSON.stringify(body),
    });
    return NextResponse.json(data);
  } catch (error) {
    return NextResponse.json(
      { success: false, error: { code: "HOTSPOT_TEMPLATE_FAILED", message: error instanceof Error ? error.message : "Gagal membuat template hotspot" } },
      { status: 502 },
    );
  }
}
