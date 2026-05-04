import { MikrotikRouterWorkspace } from "../../components/router-detail/MikrotikRouterWorkspace";

export default async function HotspotPage({ params }: { params: Promise<{ id: string }> }) {
  const { id } = await params;
  return <MikrotikRouterWorkspace routerId={id} section="hotspot" />;
}
