import { NextResponse } from "next/server";
import { networkApi } from "../../../../../../../../lib/network-api";

export async function PUT(request: Request, { params }: { params: Promise<{ id: string; assignmentId: string }> }) {
  try {
    const { id, assignmentId } = await params;
    const body = await request.json();
    const data = await networkApi(`/api/v1/mikrotik/routers/${id}/static-ip/assignments/${assignmentId}`, {
      method: "PUT",
      body: JSON.stringify(body),
    });
    return NextResponse.json(data);
  } catch (error) {
    return NextResponse.json({ success: false, error: { code: "STATIC_IP_UPDATE_FAILED", message: error instanceof Error ? error.message : "Gagal mengubah static IP" } }, { status: 502 });
  }
}

export async function DELETE(request: Request, { params }: { params: Promise<{ id: string; assignmentId: string }> }) {
  try {
    const { id, assignmentId } = await params;
    const body = await request.json();
    const data = await networkApi(`/api/v1/mikrotik/routers/${id}/static-ip/assignments/${assignmentId}`, {
      method: "DELETE",
      body: JSON.stringify(body),
    });
    return NextResponse.json(data);
  } catch (error) {
    return NextResponse.json({ success: false, error: { code: "STATIC_IP_DELETE_FAILED", message: error instanceof Error ? error.message : "Gagal menghapus static IP" } }, { status: 502 });
  }
}
