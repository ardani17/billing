import { MikrotikLiveDetailPage } from "../MikrotikLiveClient";

export default async function Page({ params }: { params: Promise<{ id: string }> }) {
  const { id } = await params;
  return <MikrotikLiveDetailPage routerId={id} />;
}
