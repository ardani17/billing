import { NextResponse } from "next/server";
import { networkApi } from "../../../../../lib/network-api";

export async function GET(_request: Request, { params }: { params: Promise<{ id: string }> }) {
  try {
    const { id } = await params;
    const data = await networkApi(`/api/v1/mikrotik/bulk-jobs/${id}`);
    return NextResponse.json(data);
  } catch (error) {
    return NextResponse.json(
      { success: false, error: { code: "BULK_JOB_GET_FAILED", message: error instanceof Error ? error.message : "Gagal mengambil detail bulk job MikroTik" } },
      { status: 502 },
    );
  }
}

