import { NextResponse } from "next/server";
import { networkApi } from "../../../../../../../lib/network-api";

export async function DELETE(_request: Request, { params }: { params: Promise<{ id: string; backupId: string }> }) {
  try {
    const { id, backupId } = await params;
    const data = await networkApi(`/api/v1/mikrotik/routers/${id}/backups/${backupId}`, { method: "DELETE" });
    return NextResponse.json(data);
  } catch (error) {
    return NextResponse.json(
      { success: false, error: { code: "BACKUP_DELETE_FAILED", message: error instanceof Error ? error.message : "Gagal menghapus backup router" } },
      { status: 502 },
    );
  }
}
