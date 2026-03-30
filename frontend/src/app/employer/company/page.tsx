import {
  type CompanyMembership,
  type EmployerProfile,
  type ListResponse,
  type VerificationRequest,
} from "@/lib/api";
import { serverFetchJson } from "@/lib/server-api";

import { CompanyCreateForm, VerificationRequestForm } from "@/components/workspace-forms";
import { EmptyState, SectionHeading, VerificationRow } from "@/components/ui";

export default async function EmployerCompanyPage() {
  const [profile, companies] = await Promise.all([
    serverFetchJson<EmployerProfile>("/employer/profile"),
    serverFetchJson<ListResponse<CompanyMembership>>("/employer/companies?page=1&pageSize=20"),
  ]);

  const companyMembership = companies.items[0];
  const verificationRequests = companyMembership
    ? await serverFetchJson<ListResponse<VerificationRequest>>(
        `/employer/companies/${companyMembership.companyId}/verification-requests?page=1&pageSize=20`,
      )
    : { items: [] };

  return (
    <div className="two-column-page">
      <section className="section-block">
        <SectionHeading
          eyebrow="Компания"
          title="Публичное лицо команды"
          text="Профиль компании влияет на доверие не меньше, чем сама карточка возможности."
        />
        {companyMembership ? (
          <div className="detail-panel detail-panel--stack">
            <h2>{companyMembership.company.brandName}</h2>
            <p>{companyMembership.company.description ?? "Описание компании ещё не заполнено."}</p>
            <div className="list-stack">
              <div className="list-row">
                <div>
                  <p className="list-row__title">Контактное лицо</p>
                  <p className="list-row__meta">{profile.fullName ?? "Не указано"}</p>
                </div>
              </div>
              <div className="list-row">
                <div>
                  <p className="list-row__title">Статус участия</p>
                  <p className="list-row__meta">{companyMembership.membershipStatus}</p>
                </div>
              </div>
            </div>
          </div>
        ) : (
          <EmptyState
            title="Компания ещё не создана"
            text="Сначала создаём базовый профиль компании, потом к нему можно привязывать карточки и запросы на проверку."
          />
        )}
      </section>

      <aside className="side-stack">
        {!companyMembership ? (
          <section className="detail-panel">
            <h2>Создать компанию</h2>
            <CompanyCreateForm />
          </section>
        ) : (
          <>
            <section className="detail-panel">
              <h2>Отправить верификацию</h2>
              <VerificationRequestForm companyId={companyMembership.companyId} />
            </section>
            <section className="detail-panel">
              <h2>История проверок</h2>
              {verificationRequests.items.length > 0 ? (
                <div className="list-stack">
                  {verificationRequests.items.map((item) => (
                    <VerificationRow key={item.id} item={item} />
                  ))}
                </div>
              ) : (
                <p>Запросы на верификацию ещё не отправлялись.</p>
              )}
            </section>
          </>
        )}
      </aside>
    </div>
  );
}
