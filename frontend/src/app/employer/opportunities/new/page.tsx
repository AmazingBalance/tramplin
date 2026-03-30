import {
  type CompanyMembership,
  type ListResponse,
} from "@/lib/api";
import { serverFetchJson } from "@/lib/server-api";

import { OpportunityCreateForm } from "@/components/workspace-forms";
import { SectionHeading } from "@/components/ui";

export default async function EmployerNewOpportunityPage() {
  const companies = await serverFetchJson<ListResponse<CompanyMembership>>(
    "/employer/companies?page=1&pageSize=20",
  );

  return (
    <div className="two-column-page">
      <section className="section-block">
        <SectionHeading
          eyebrow="Новая карточка"
          title="Создай сильную карточку без перегруза"
          text="Форма разбита на минимально нужные поля: компания, смысл, формат, локация и деньги."
        />
        <OpportunityCreateForm companies={companies.items} />
      </section>
    </div>
  );
}
