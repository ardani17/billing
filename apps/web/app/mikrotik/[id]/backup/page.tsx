import { MikrotikRouterWorkspace } from "../../components/router-detail/MikrotikRouterWorkspace";

export default async function BackupPage({ params }: { params: Promise<{ id: string }> }) {
  const { id } = await params;
  return <MikrotikRouterWorkspace routerId={id} section="backup" />;
}
