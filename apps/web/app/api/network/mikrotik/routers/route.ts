import { NextResponse } from "next/server";
import { networkApi } from "../../../../lib/network-api";

export async function GET() {
  try {
    const data = await networkApi("/api/v1/mikrotik/routers?page_size=50");
    return NextResponse.json(data);
  } catch (error) {
    return NextResponse.json(
      {
        success: false,
        error: {
          code: "NETWORK_SERVICE_UNAVAILABLE",
          message: error instanceof Error ? error.message : "Network service tidak tersedia",
        },
      },
      { status: 502 },
    );
  }
}

export async function POST(request: Request) {
  try {
    const body = await request.json();
    const data = await networkApi("/api/v1/mikrotik/routers", {
      method: "POST",
      body: JSON.stringify(body),
    });
    return NextResponse.json(data);
  } catch (error) {
    return NextResponse.json(
      {
        success: false,
        error: {
          code: "NETWORK_SERVICE_ERROR",
          message: error instanceof Error ? error.message : "Gagal menyimpan router",
        },
      },
      { status: 502 },
    );
  }
}
