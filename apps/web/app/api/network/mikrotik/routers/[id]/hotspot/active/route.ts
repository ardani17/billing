import { NextResponse } from "next/server";
import { networkApi } from "../../../../../../../lib/network-api";

export async function GET(_request: Request, { params }: { params: Promise<{ id: string }> }) {
  try {
    const { id } = await params;
    const data = await networkApi(`/api/v1/mikrotik/routers/${id}/hotspot/active`);
    return NextResponse.json(data);
  } catch (error) {
    return NextResponse.json(
      { success: false, error: { code: "HOTSPOT_ACTIVE_FAILED", message: error instanceof Error ? error.message : "Gagal mengambil session hotspot" } },
      { status: 502 },
    );
  }
}
