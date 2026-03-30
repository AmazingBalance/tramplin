import Link from "next/link";

import {
  type ListResponse,
  type Opportunity,
} from "@/lib/api";
import { serverFetchJson } from "@/lib/server-api";

import { EmptyState, OpportunityCard, SectionHeading } from "@/components/ui";

export default async function EmployerOpportunitiesPage() {
  const opportunities = await serverFetchJson<ListResponse<Opportunity>>(
    "/employer/opportunities?page=1&pageSize=20",
  );

  return (
    <section className="section-block">
      <SectionHeading
        eyebrow="Карточки"
        title="Лента всех возможностей команды"
        text="Здесь видно, что активно, что ждёт публикации, а что пока остаётся черновиком."
        action={
          <Link href="/employer/opportunities/new" className="button button--primary">
            Создать карточку
          </Link>
        }
      />
      {opportunities.items.length > 0 ? (
        <div className="company-grid">
          {opportunities.items.map((item) => (
            <OpportunityCard
              key={item.id}
              opportunity={item}
              actions={
                <div className="card-action-row">
                  <Link href={`/opportunities/${item.slug}`} className="button button--ghost">
                    Публичный вид
                  </Link>
                </div>
              }
            />
          ))}
        </div>
      ) : (
        <EmptyState
          title="Карточек пока нет"
          text="Создай первую возможность, чтобы открыть ленту кандидатов и начать набор."
          action={
            <Link href="/employer/opportunities/new" className="button button--primary">
              Создать карточку
            </Link>
          }
        />
      )}
    </section>
  );
}
