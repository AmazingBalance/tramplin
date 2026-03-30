import Link from "next/link";

import {
  type Company,
  type ListResponse,
  type SavedOpportunity,
} from "@/lib/api";
import { serverFetchJson } from "@/lib/server-api";

import { CompanyCard, EmptyState, OpportunityCard, SectionHeading } from "@/components/ui";

export default async function ApplicantSavedPage() {
  const [savedOpportunities, savedCompanies] = await Promise.all([
    serverFetchJson<ListResponse<SavedOpportunity>>("/me/saved-opportunities?page=1&pageSize=20"),
    serverFetchJson<ListResponse<{ company: Company; companyId: string }>>(
      "/me/saved-companies?page=1&pageSize=20",
    ),
  ]);

  return (
    <div className="stack-page">
      <section className="section-block">
        <SectionHeading
          eyebrow="Сохранённые возможности"
          title="Личный shortlist"
          text="Здесь собраны позиции, к которым хочется вернуться без повторного поиска."
        />
        {savedOpportunities.items.length > 0 ? (
          <div className="company-grid">
            {savedOpportunities.items.map((item) => (
              <OpportunityCard key={item.opportunityId} opportunity={item.opportunity} />
            ))}
          </div>
        ) : (
          <EmptyState
            title="Пока пусто"
            text="Сохрани пару карточек на главной, и здесь появится твой shortlist."
            action={
              <Link href="/" className="button button--primary">
                Вернуться в каталог
              </Link>
            }
          />
        )}
      </section>

      <section className="section-block">
        <SectionHeading
          eyebrow="Сохранённые компании"
          title="Команды, к которым хочется присмотреться"
        />
        {savedCompanies.items.length > 0 ? (
          <div className="company-grid">
            {savedCompanies.items.map((item) => (
              <CompanyCard key={item.companyId} company={item.company} />
            ))}
          </div>
        ) : (
          <EmptyState
            title="Компаний пока нет"
            text="Сохраняй работодателей на главной или в их профиле, чтобы не потерять ориентиры."
          />
        )}
      </section>
    </div>
  );
}
