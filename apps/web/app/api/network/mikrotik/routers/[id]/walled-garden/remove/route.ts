import { NextResponse } from "next/server";
import { networkApi } from "../../../../../../../lib/network-api";

export async function POST(_request: Request, { params }: { params: Promise<{ id: string }> }) {
  try {
    const { id } = await params;
    const data = await networkApi(`/api/v1/mikrotik/routers/${id}/walled-garden/remove`, {
      method: "POST",
    });
    return NextResponse.json(data);
  } catch (error) {
    return NextResponse.json(
      {
        success: false,
        error: {
          code: "WALLED_GARDEN_REMOVE_FAILED",
          message: error instanceof Error ? error.message : "Gagal menghapus rule walled garden",
        },
      },
      { status: 502 },
    );
  }
}
