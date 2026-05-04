import { NextResponse } from "next/server";
import { networkApi } from "../../../../lib/network-api";

export async function GET(request: Request) {
  try {
    const url = new URL(request.url);
    const data = await networkApi(`/api/v1/mikrotik/bulk-jobs${url.search}`);
    return NextResponse.json(data);
  } catch (error) {
    return NextResponse.json(
      { success: false, error: { code: "BULK_JOB_LIST_FAILED", message: error instanceof Error ? error.message : "Gagal mengambil bulk job MikroTik" } },
      { status: 502 },
    );
  }
}

export async function POST(request: Request) {
  try {
    const body = await request.json();
    const data = await networkApi("/api/v1/mikrotik/bulk-jobs", {
      method: "POST",
      body: JSON.stringify(body),
    });
    return NextResponse.json(data);
  } catch (error) {
    return NextResponse.json(
      { success: false, error: { code: "BULK_JOB_CREATE_FAILED", message: error instanceof Error ? error.message : "Gagal menjalankan bulk action MikroTik" } },
      { status: 502 },
    );
  }
}

