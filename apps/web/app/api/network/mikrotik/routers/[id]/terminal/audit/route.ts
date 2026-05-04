import { NextResponse } from "next/server";
import { networkApi } from "../../../../../../../lib/network-api";

export async function GET(request: Request, { params }: { params: Promise<{ id: string }> }) {
  try {
    const { id } = await params;
    const url = new URL(request.url);
    const data = await networkApi(`/api/v1/mikrotik/routers/${id}/terminal/audit${url.search}`);
    return NextResponse.json(data);
  } catch (error) {
    return NextResponse.json(
      {
        success: false,
        error: {
          code: "TERMINAL_AUDIT_FAILED",
          message: error instanceof Error ? error.message : "Gagal mengambil audit terminal MikroTik",
        },
      },
      { status: 502 },
    );
  }
}
