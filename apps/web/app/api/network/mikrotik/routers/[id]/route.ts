import { NextResponse } from "next/server";
import { networkApi } from "../../../../../lib/network-api";

export async function GET(_request: Request, { params }: { params: Promise<{ id: string }> }) {
  try {
    const { id } = await params;
    const data = await networkApi(`/api/v1/mikrotik/routers/${id}`);
    return NextResponse.json(data);
  } catch (error) {
    return NextResponse.json(
      {
        success: false,
        error: {
          code: "NETWORK_SERVICE_ERROR",
          message: error instanceof Error ? error.message : "Gagal mengambil router",
        },
      },
      { status: 502 },
    );
  }
}

export async function PUT(request: Request, { params }: { params: Promise<{ id: string }> }) {
  try {
    const { id } = await params;
    const body = await request.json();
    const data = await networkApi(`/api/v1/mikrotik/routers/${id}`, {
      method: "PUT",
      body: JSON.stringify(body),
    });
    return NextResponse.json(data);
  } catch (error) {
    return NextResponse.json(
      {
        success: false,
        error: {
          code: "ROUTER_UPDATE_FAILED",
          message: error instanceof Error ? error.message : "Gagal mengubah router",
        },
      },
      { status: 502 },
    );
  }
}

export async function DELETE(_request: Request, { params }: { params: Promise<{ id: string }> }) {
  try {
    const { id } = await params;
    await networkApi(`/api/v1/mikrotik/routers/${id}`, {
      method: "DELETE",
    });
    return NextResponse.json({ success: true });
  } catch (error) {
    return NextResponse.json(
      {
        success: false,
        error: {
          code: "ROUTER_DELETE_FAILED",
          message: error instanceof Error ? error.message : "Gagal menghapus router",
        },
      },
      { status: 502 },
    );
  }
}
