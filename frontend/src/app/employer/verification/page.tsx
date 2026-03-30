import {
  type CompanyMembership,
  type ListResponse,
  type VerificationRequest,
} from "@/lib/api";
import { serverFetchJson } from "@/lib/server-api";

import { VerificationRequestForm } from "@/components/workspace-forms";
import { EmptyState, SectionHeading, VerificationRow } from "@/components/ui";

export default async function EmployerVerificationPage() {
  const companies = await serverFetchJson<ListResponse<CompanyMembership>>(
    "/employer/companies?page=1&pageSize=20",
  );

  const companyMembership = companies.items[0];
  const requests = companyMembership
    ? await serverFetchJson<ListResponse<VerificationRequest>>(
        `/employer/companies/${companyMembership.companyId}/verification-requests?page=1&pageSize=20`,
      )
    : { items: [] };

  return (
    <div className="two-column-page">
      <section className="section-block">
        <SectionHeading
          eyebrow="Верификация"
          title="Статус доверия должен быть прозрачным"
          text="Здесь собирается история заявок на проверку, чтобы работодатель понимал текущий статус без похода по разным экранам."
        />
        {requests.items.length > 0 ? (
          <div className="list-stack">
            {requests.items.map((item) => (
              <VerificationRow key={item.id} item={item} />
            ))}
          </div>
        ) : (
          <EmptyState
            title="Запросов пока нет"
            text="Как только отправишь первый запрос, здесь появится статус и комментарии куратора."
          />
        )}
      </section>

      <aside className="side-stack">
        {companyMembership ? (
          <section className="detail-panel">
            <h2>Новый запрос</h2>
            <VerificationRequestForm companyId={companyMembership.companyId} />
          </section>
        ) : null}
      </aside>
    </div>
  );
}
