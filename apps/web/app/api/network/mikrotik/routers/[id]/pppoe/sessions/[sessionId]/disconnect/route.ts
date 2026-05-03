import { NextResponse } from "next/server";
import { networkApi } from "../../../../../../../../../lib/network-api";

export async function POST(_request: Request, { params }: { params: Promise<{ id: string; sessionId: string }> }) {
  try {
    const { id, sessionId } = await params;
    const data = await networkApi(`/api/v1/mikrotik/routers/${id}/pppoe/sessions/${sessionId}/disconnect`, {
      method: "POST",
    });
    return NextResponse.json(data);
  } catch (error) {
    return NextResponse.json(
      {
        success: false,
        error: {
          code: "PPPOE_SESSION_DISCONNECT_FAILED",
          message: error instanceof Error ? error.message : "Gagal memutus session PPPoE",
        },
      },
      { status: 502 },
    );
  }
}
