import { NextResponse } from "next/server";
import { networkApi } from "../../../../../../../../lib/network-api";

export async function PUT(request: Request, { params }: { params: Promise<{ id: string; bindingId: string }> }) {
  try {
    const { id, bindingId } = await params;
    const body = await request.json();
    const data = await networkApi(`/api/v1/mikrotik/routers/${id}/dhcp/bindings/${bindingId}`, {
      method: "PUT",
      body: JSON.stringify(body),
    });
    return NextResponse.json(data);
  } catch (error) {
    return NextResponse.json({ success: false, error: { code: "DHCP_BINDING_UPDATE_FAILED", message: error instanceof Error ? error.message : "Gagal mengubah DHCP binding" } }, { status: 502 });
  }
}

export async function DELETE(request: Request, { params }: { params: Promise<{ id: string; bindingId: string }> }) {
  try {
    const { id, bindingId } = await params;
    const body = await request.json();
    const data = await networkApi(`/api/v1/mikrotik/routers/${id}/dhcp/bindings/${bindingId}`, {
      method: "DELETE",
      body: JSON.stringify(body),
    });
    return NextResponse.json(data);
  } catch (error) {
    return NextResponse.json({ success: false, error: { code: "DHCP_BINDING_DELETE_FAILED", message: error instanceof Error ? error.message : "Gagal menghapus DHCP binding" } }, { status: 502 });
  }
}
