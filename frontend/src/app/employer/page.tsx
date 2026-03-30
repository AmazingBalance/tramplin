import Link from "next/link";

import {
  type CompanyMembership,
  type EmployerProfile,
  type ListResponse,
  type Opportunity,
} from "@/lib/api";
import { serverFetchJson } from "@/lib/server-api";

import { PlatformPulse } from "@/components/platform-pulse";
import { MetricCard, QueueCard, SectionHeading } from "@/components/ui";

export default async function EmployerDashboardPage() {
  const [profile, companies, opportunities] = await Promise.all([
    serverFetchJson<EmployerProfile>("/employer/profile"),
    serverFetchJson<ListResponse<CompanyMembership>>("/employer/companies?page=1&pageSize=20"),
    serverFetchJson<ListResponse<Opportunity>>("/employer/opportunities?page=1&pageSize=20"),
  ]);

  const activeCount = opportunities.items.filter((item) => item.status === "active").length;
  const draftCount = opportunities.items.filter((item) => item.status === "draft").length;

  return (
    <>
      <section className="dashboard-hero">
        <div className="dashboard-hero__copy">
          <p className="eyebrow">Главная</p>
          <h2>Управляйте потоком кандидатов без лишнего шума.</h2>
          <p>
            На главной роли собраны поиск по рынку, состояние ваших карточек и ближайшие действия
            по компании.
          </p>
          <div className="dashboard-hero__actions">
            <Link href="/employer/opportunities/new" className="button button--primary">
              Создать карточку
            </Link>
            <Link href="/employer/company" className="button button--ghost">
              Открыть компанию
            </Link>
          </div>
        </div>
        <div className="dashboard-hero__aside">
          <div className="pulse-card pulse-card--accent">
            <strong>{companies.items[0]?.company.brandName ?? "Компания"}</strong>
            <p>Основной рабочий контур: верификация, карточки возможностей и отклики.</p>
          </div>
          <div className="pulse-card">
            <strong>Контактное лицо</strong>
            <p>{profile.fullName ?? "Добавьте ФИО и должность, чтобы карточка компании выглядела убедительнее."}</p>
          </div>
        </div>
      </section>

      <PlatformPulse
        role="employer"
        title="Смотрите рынок так, как его видит кандидат"
        text="Лента и карта помогают быстро сравнить вашу карточку с остальной выдачей платформы."
      />

      <section className="dashboard-metrics">
        <MetricCard label="Компании" value={companies.items.length} tone="accent" />
        <MetricCard label="Активные карточки" value={activeCount} />
        <MetricCard label="Черновики" value={draftCount} />
        <MetricCard label="Контактное лицо" value={profile.fullName ?? "Не указано"} tone="soft" />
      </section>

      <section className="section-block">
        <SectionHeading
          eyebrow="Очередь действий"
          title="Главное в работе команды сейчас"
          action={
            <Link href="/employer/opportunities/new" className="button button--primary">
              Создать новую карточку
            </Link>
          }
        />
        <div className="queue-grid">
          <QueueCard
            title="Профиль компании"
            subtitle="Проверь базовую информацию и убедись, что компания выглядит убедительно."
            href="/employer/company"
          />
          <QueueCard
            title="Верификация"
            subtitle="Если проверка ещё не отправлена или требует доработки, это должно быть видно сразу."
            href="/employer/verification"
          />
          <QueueCard
            title="Отклики"
            subtitle="Следи за кандидатами по каждой карточке и не теряй темп ответа."
            href="/employer/applicants"
          />
        </div>
      </section>
    </>
  );
}
