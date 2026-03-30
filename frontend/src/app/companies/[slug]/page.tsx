import Link from "next/link";
import { notFound } from "next/navigation";

import {
  composeLocation,
  type Company,
  type ListResponse,
  type Opportunity,
} from "@/lib/api";
import { getCurrentUserSafe, serverFetchJson, serverFetchJsonSafe } from "@/lib/server-api";

import { CompanyMark } from "@/components/company-mark";
import { SiteHeader } from "@/components/site-header";
import { OpportunityCard, SectionHeading, StatusBadge } from "@/components/ui";
import { SaveCompanyButton } from "@/components/user-actions";

export default async function CompanyPage({
  params,
}: {
  params: Promise<{ slug: string }>;
}) {
  const { slug } = await params;
  const [user, company, opportunities] = await Promise.all([
    getCurrentUserSafe(),
    serverFetchJsonSafe<Company>(`/public/companies/by-slug/${slug}`),
    serverFetchJson<ListResponse<Opportunity>>("/public/opportunities?page=1&pageSize=60"),
  ]);

  if (!company) {
    notFound();
  }

  const companyOpportunities = opportunities.items.filter(
    (item) => item.company.slug === company.slug,
  );

  return (
    <div className="page-shell">
      <SiteHeader user={user} />
      <main className="detail-page">
        <section className="detail-hero">
          <div className="detail-hero__copy">
            <CompanyMark company={company} size="full" />
            <p className="eyebrow">Профиль компании</p>
            <h1>{company.brandName}</h1>
            <p>{company.description ?? "Компания оформляет профиль, но уже присутствует в каталоге."}</p>
            <dl className="detail-grid">
              <div>
                <dt>Сфера</dt>
                <dd>{company.industry ?? "Уточняется"}</dd>
              </div>
              <div>
                <dt>Локация</dt>
                <dd>{composeLocation(company.headquartersLocation)}</dd>
              </div>
              <div>
                <dt>Сайт</dt>
                <dd>{company.websiteUrl ?? "Не указан"}</dd>
              </div>
              <div>
                <dt>Статус</dt>
                <dd>
                  <StatusBadge value={company.verificationStatus} />
                </dd>
              </div>
            </dl>
            <div className="detail-links">
              {company.socialLinks?.map((item) => (
                <Link key={item.id} href={item.url} className="inline-link">
                  {item.platform}
                </Link>
              ))}
            </div>
          </div>
          <aside className="detail-hero__aside">
            <div className="detail-panel">
              <h2>Быстрое действие</h2>
              <p>Сохрани компанию, чтобы вернуться к ней позже и увидеть её на карте и в подборках.</p>
              <SaveCompanyButton companyId={company.id} currentRole={user?.role ?? null} />
            </div>
          </aside>
        </section>

        <section className="section-block">
          <SectionHeading
            eyebrow="Открытые направления"
            title="То, что компания уже вынесла в каталог"
            text="Карточки ниже можно открывать, сохранять и добавлять в личную воронку."
          />
          <div className="company-grid">
            {companyOpportunities.map((item) => (
              <OpportunityCard key={item.id} opportunity={item} />
            ))}
          </div>
        </section>
      </main>
    </div>
  );
}
