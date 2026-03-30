import { type UserRole, type Company, type ListResponse, type Opportunity, type Tag } from "@/lib/api";
import { serverFetchJson } from "@/lib/server-api";

import { HomeExplorer } from "@/components/home-explorer";
import { SectionHeading } from "@/components/ui";

export async function PlatformPulse({
  role,
  title = "Поиск по платформе",
  text = "Вакансии, стажировки, менторские программы и события доступны прямо с главного экрана роли.",
}: {
  role: UserRole;
  title?: string;
  text?: string;
}) {
  const [opportunities, companies, tags] = await Promise.all([
    serverFetchJson<ListResponse<Opportunity>>("/public/opportunities?page=1&pageSize=60"),
    serverFetchJson<ListResponse<Company>>("/public/companies?page=1&pageSize=24"),
    serverFetchJson<ListResponse<Tag> | { items: Tag[] }>("/public/tags"),
  ]);

  const tagItems = "meta" in tags ? tags.items : tags.items;

  return (
    <section className="section-block">
      <SectionHeading eyebrow="Поиск" title={title} text={text} />
      <HomeExplorer
        opportunities={opportunities.items}
        companies={companies.items}
        tags={tagItems}
        currentRole={role}
      />
    </section>
  );
}
