import { NextResponse } from "next/server";
import { networkApi } from "../../../../../lib/network-api";

export async function GET() {
  try {
    const data = await networkApi("/api/v1/mikrotik/status/summary");
    return NextResponse.json(data);
  } catch (error) {
    return NextResponse.json(
      {
        success: false,
        error: {
          code: "NETWORK_SERVICE_ERROR",
          message: error instanceof Error ? error.message : "Gagal mengambil summary MikroTik",
        },
      },
      { status: 502 },
    );
  }
}
