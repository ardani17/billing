import { NextResponse } from "next/server";
import { networkApi } from "../../../../../../../../lib/network-api";

export async function DELETE(_request: Request, { params }: { params: Promise<{ id: string; userId: string }> }) {
  try {
    const { id, userId } = await params;
    const data = await networkApi(`/api/v1/mikrotik/routers/${id}/pppoe/users/${userId}`, {
      method: "DELETE",
    });
    return NextResponse.json(data);
  } catch (error) {
    return NextResponse.json(
      {
        success: false,
        error: {
          code: "PPPOE_USER_DELETE_FAILED",
          message: error instanceof Error ? error.message : "Gagal menghapus PPPoE user",
        },
      },
      { status: 502 },
    );
  }
}
