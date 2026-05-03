import { NextResponse } from "next/server";
import { networkApi } from "../../../../../../../../../lib/network-api";

export async function POST(_request: Request, { params }: { params: Promise<{ id: string; userId: string }> }) {
  try {
    const { id, userId } = await params;
    const data = await networkApi(`/api/v1/mikrotik/routers/${id}/pppoe/users/${userId}/disconnect`, {
      method: "POST",
    });
    return NextResponse.json(data);
  } catch (error) {
    return NextResponse.json(
      {
        success: false,
        error: {
          code: "PPPOE_USER_DISCONNECT_FAILED",
          message: error instanceof Error ? error.message : "Gagal memutus PPPoE user",
        },
      },
      { status: 502 },
    );
  }
}
