import { NextResponse } from "next/server";
import { networkApi } from "../../../../../../../../../lib/network-api";

export async function POST(_request: Request, { params }: { params: Promise<{ id: string; assignmentId: string }> }) {
  try {
    const { id, assignmentId } = await params;
    const data = await networkApi(`/api/v1/mikrotik/routers/${id}/static-ip/assignments/${assignmentId}/unisolate`, { method: "POST" });
    return NextResponse.json(data);
  } catch (error) {
    return NextResponse.json({ success: false, error: { code: "STATIC_IP_UNISOLATE_FAILED", message: error instanceof Error ? error.message : "Gagal buka isolir static IP" } }, { status: 502 });
  }
}
