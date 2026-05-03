import { InvoiceDetailLivePage } from "../../components/real-pages";

export default async function Page({ params }: { params: Promise<{ id: string }> }) {
  const { id } = await params;
  return <InvoiceDetailLivePage id={id} />;
}
