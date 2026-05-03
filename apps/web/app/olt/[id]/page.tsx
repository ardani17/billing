import { OltDetailLivePage } from "../../components/real-pages";

export default async function Page({ params }: { params: Promise<{ id: string }> }) {
  const { id } = await params;
  return <OltDetailLivePage id={id} />;
}
