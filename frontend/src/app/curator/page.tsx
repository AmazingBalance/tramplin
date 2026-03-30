import Link from "next/link";

import {
  type ListResponse,
  type ModerationCase,
  type VerificationRequest,
} from "@/lib/api";
import { serverFetchJson } from "@/lib/server-api";

import { PlatformPulse } from "@/components/platform-pulse";
import { MetricCard, QueueCard, SectionHeading } from "@/components/ui";

export default async function CuratorDashboardPage() {
  const [verificationRequests, moderationCases] = await Promise.all([
    serverFetchJson<ListResponse<VerificationRequest>>(
      "/curator/verification-requests?page=1&pageSize=20",
    ),
    serverFetchJson<ListResponse<ModerationCase>>("/curator/moderation-cases?page=1&pageSize=20"),
  ]);
  const pendingTotal =
    verificationRequests.items.filter((item) => item.status === "pending").length +
    moderationCases.items.filter((item) => item.status === "pending").length;

  return (
    <>
      <section className="dashboard-hero">
        <div className="dashboard-hero__copy">
          <p className="eyebrow">Главная</p>
          <h2>Держите качество платформы под контролем.</h2>
          <p>
            Главная куратора сводит в одно место проверки, модерацию и публичную поверхность,
            которую видит пользователь.
          </p>
          <div className="dashboard-hero__actions">
            <Link href="/curator/verification" className="button button--primary">
              Открыть проверки
            </Link>
            <Link href="/curator/moderation" className="button button--ghost">
              Перейти к модерации
            </Link>
          </div>
        </div>
        <div className="dashboard-hero__aside">
          <div className="pulse-card pulse-card--accent">
            <strong>{pendingTotal} задач требуют решения</strong>
            <p>Приоритет на старте: неподтверждённые компании и карточки со статусом pending.</p>
          </div>
          <div className="pulse-card">
            <strong>Общий обзор</strong>
            <p>Сначала разберите очередь, затем проверьте, как изменения выглядят на витрине.</p>
          </div>
        </div>
      </section>

      <PlatformPulse
        role="curator"
        title="Публичная выдача платформы"
        text="Так видит рынок пользователь: карта, лента, теги и сохранённые компании."
      />

      <section className="dashboard-metrics">
        <MetricCard label="Проверки" value={verificationRequests.items.length} tone="accent" />
        <MetricCard label="Модерация" value={moderationCases.items.length} />
        <MetricCard
          label="Требуют решения"
          value={pendingTotal}
          tone="soft"
        />
      </section>

      <section className="section-block">
        <SectionHeading
          eyebrow="Рабочие очереди"
          title="Что нужно разобрать в первую очередь"
          action={
            <Link href="/curator/verification" className="button button--ghost">
              Открыть проверки
            </Link>
          }
        />
        <div className="queue-grid">
          <QueueCard
            title="Проверки компаний"
            subtitle="Компании, которые ждут подтверждения или комментария от куратора."
            href="/curator/verification"
          />
          <QueueCard
            title="Модерация карточек"
            subtitle="Карточки и сущности, которым нужны изменения, блокировка или подтверждение."
            href="/curator/moderation"
          />
          <QueueCard
            title="Пользователи"
            subtitle="Быстрый доступ к applicant и employer профилям для правок и проверки статусов."
            href="/curator/users"
          />
        </div>
      </section>
    </>
  );
}
