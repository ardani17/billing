import { NextResponse } from "next/server";
import { networkApi } from "../../../../../../../lib/network-api";

export async function POST(request: Request, { params }: { params: Promise<{ id: string }> }) {
  try {
    const { id } = await params;
    const body = await request.json();
    const data = await networkApi(`/api/v1/mikrotik/routers/${id}/terminal/execute`, {
      method: "POST",
      body: JSON.stringify(body),
    });
    return NextResponse.json(data);
  } catch (error) {
    return NextResponse.json(
      {
        success: false,
        error: {
          code: "TERMINAL_EXECUTE_FAILED",
          message: error instanceof Error ? error.message : "Gagal menjalankan terminal MikroTik",
        },
      },
      { status: 502 },
    );
  }
}
