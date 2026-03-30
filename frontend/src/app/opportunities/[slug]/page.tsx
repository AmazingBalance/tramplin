import Link from "next/link";
import { notFound } from "next/navigation";

import {
  composeLocation,
  formatDate,
  formatMoney,
  formatOpportunityType,
  formatParticipation,
  type ListResponse,
  type Opportunity,
} from "@/lib/api";
import { getCurrentUserSafe, serverFetchJson, serverFetchJsonSafe } from "@/lib/server-api";

import { SiteHeader } from "@/components/site-header";
import { OpportunityCard, SectionHeading, TagStrip } from "@/components/ui";
import { ApplicationComposer, SaveOpportunityButton } from "@/components/user-actions";

export default async function OpportunityPage({
  params,
}: {
  params: Promise<{ slug: string }>;
}) {
  const { slug } = await params;
  const [user, opportunity, related] = await Promise.all([
    getCurrentUserSafe(),
    serverFetchJsonSafe<Opportunity>(`/public/opportunities/by-slug/${slug}`),
    serverFetchJson<ListResponse<Opportunity>>("/public/opportunities?page=1&pageSize=60"),
  ]);

  if (!opportunity) {
    notFound();
  }

  const relatedItems = related.items.filter((item) => item.id !== opportunity.id).slice(0, 2);

  return (
    <div className="page-shell">
      <SiteHeader user={user} />
      <main className="detail-page">
        <section className="detail-hero">
          <div className="detail-hero__copy">
            <p className="eyebrow">{formatOpportunityType(opportunity.type)}</p>
            <h1>{opportunity.title}</h1>
            <p>{opportunity.description ?? opportunity.summary}</p>
            <TagStrip tags={opportunity.tags} />
            <dl className="detail-grid">
              <div>
                <dt>Компания</dt>
                <dd>{opportunity.company.brandName}</dd>
              </div>
              <div>
                <dt>Формат</dt>
                <dd>{formatParticipation(opportunity.participationFormat)}</dd>
              </div>
              <div>
                <dt>Локация</dt>
                <dd>{composeLocation(opportunity.location)}</dd>
              </div>
              <div>
                <dt>Доход</dt>
                <dd>{formatMoney(opportunity.vacancyDetails)}</dd>
              </div>
              <div>
                <dt>Публикация</dt>
                <dd>{formatDate(opportunity.publishedAt)}</dd>
              </div>
              <div>
                <dt>Дедлайн</dt>
                <dd>{formatDate(opportunity.expiresAt)}</dd>
              </div>
            </dl>
          </div>
          <aside className="detail-hero__aside">
            <div className="detail-panel">
              <h2>Следующий шаг</h2>
              <p>Можно сохранить возможность, а можно сразу отправить отклик с коротким сообщением.</p>
              <SaveOpportunityButton opportunityId={opportunity.id} currentRole={user?.role ?? null} />
              <ApplicationComposer opportunityId={opportunity.id} currentRole={user?.role ?? null} />
            </div>
            <div className="detail-panel">
              <h2>Контакты</h2>
              <p>{opportunity.contactEmail ?? "Контактный email появится позже"}</p>
              <p>{opportunity.contactPhone ?? "Телефон не указан"}</p>
              {opportunity.links?.map((item) => (
                <Link key={item.id} href={item.url} className="inline-link">
                  {item.title}
                </Link>
              ))}
            </div>
          </aside>
        </section>

        <section className="section-block">
          <SectionHeading
            eyebrow="О компании"
            title={opportunity.company.brandName}
            text={opportunity.company.description ?? "Компания уже открыла карточку, но ещё дополняет детали."}
          />
          <Link href={`/companies/${opportunity.company.slug}`} className="button button--ghost">
            Перейти в профиль компании
          </Link>
        </section>

        <section className="section-block">
          <SectionHeading
            eyebrow="Похожие возможности"
            title="Если хочется сравнить варианты, вот ещё несколько направлений."
          />
          <div className="company-grid">
            {relatedItems.map((item) => (
              <OpportunityCard key={item.id} opportunity={item} />
            ))}
          </div>
        </section>
      </main>
    </div>
  );
}
