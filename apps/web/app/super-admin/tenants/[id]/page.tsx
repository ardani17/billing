import { SuperAdminTenantDetailPage } from "../../../components/super-admin-pages";

type PageProps = {
  params: Promise<{ id: string }>;
};

export default async function Page({ params }: PageProps) {
  const { id } = await params;
  return <SuperAdminTenantDetailPage tenantId={id} />;
}
