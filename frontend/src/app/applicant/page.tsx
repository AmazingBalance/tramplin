import Link from "next/link";
import type { CSSProperties } from "react";

import {
  type ApplicantPrivacy,
  type ApplicantProfile,
  type Application,
  type Connection,
  type ListResponse,
  type SavedOpportunity,
} from "@/lib/api";
import { serverFetchJson } from "@/lib/server-api";

import { PlatformPulse } from "@/components/platform-pulse";
import { ApplicationRow, MetricCard, QueueCard, SectionHeading } from "@/components/ui";

export default async function ApplicantDashboardPage() {
  const [profile, applications, savedOpportunities, connections, privacy] = await Promise.all([
    serverFetchJson<ApplicantProfile>("/me/applicant-profile"),
    serverFetchJson<ListResponse<Application>>("/me/applications?page=1&pageSize=20"),
    serverFetchJson<ListResponse<SavedOpportunity>>("/me/saved-opportunities?page=1&pageSize=20"),
    serverFetchJson<ListResponse<Connection>>("/me/connections?page=1&pageSize=20"),
    serverFetchJson<ApplicantPrivacy>("/me/applicant/privacy"),
  ]);

  const profileFields = [
    profile.firstName,
    profile.lastName,
    profile.universityName,
    profile.programName,
    profile.studyYear,
    profile.graduationYear,
    profile.city,
    profile.about,
    profile.resumeMediaId,
  ];
  const completion = Math.round(
    (profileFields.filter((value) => value !== null && value !== "").length / profileFields.length) *
      100,
  );
  const visibilityLabel =
    privacy.profileVisibility === "public"
      ? "Открыт"
      : privacy.profileVisibility === "contacts_only"
        ? "Контакты"
        : "Приватный";

  return (
    <>
      <section className="dashboard-hero">
        <div className="dashboard-hero__copy">
          <p className="eyebrow">Главная</p>
          <h2>Подберите следующий сильный шаг.</h2>
          <p>
            Здесь начинается рабочая главная: поиск по вакансиям, стажировкам, событиям и
            менторским программам уже внутри кабинета.
          </p>
          <div className="dashboard-hero__actions">
            <Link href="/applicant/profile" className="button button--primary">
              Дополнить профиль
            </Link>
            <Link href="/applicant/applications" className="button button--ghost">
              История откликов
            </Link>
          </div>
        </div>
        <div className="dashboard-hero__aside">
          <div className="progress-wheel" style={{ "--progress": `${completion}%` } as CSSProperties}>
            <strong>{completion}%</strong>
            <span>профиль собран</span>
          </div>
          <div className="pulse-card">
            <strong>Сегодняшний фокус</strong>
            <p>Дополни профиль и открой ещё пару карточек, чтобы не потерять темп.</p>
          </div>
        </div>
      </section>

      <PlatformPulse
        role="applicant"
        title="Поиск по рынку возможностей"
        text="Выбери запрос, уточни формат, теги и доход, а затем переключайся между лентой и картой."
      />

      <section className="dashboard-metrics">
        <MetricCard label="Отклики" value={applications.items.length} tone="accent" />
        <MetricCard label="Сохранённые позиции" value={savedOpportunities.items.length} />
        <MetricCard label="Контакты" value={connections.items.length} />
        <MetricCard label="Видимость профиля" value={visibilityLabel} tone="soft" />
      </section>

      <section className="section-block">
        <SectionHeading
          eyebrow="Следующий шаг"
          title={`Сейчас важнее всего добить профиль ${profile.firstName ?? "соискателя"} и удержать темп по откликам.`}
          text="Сначала профиль и сохранённое, затем отклики и контакты. Так маршрут читается быстрее и не распадается на мелкие задачи."
          action={
            <Link href="/applicant/profile" className="button button--primary">
              Открыть профиль
            </Link>
          }
        />
        <div className="queue-grid">
          <QueueCard
            title="Профиль"
            subtitle="Добавь детали об образовании, опыте и карьерных интересах."
            href="/applicant/profile"
          />
          <QueueCard
            title="Сохранённое"
            subtitle="Проверь интересные карточки и реши, по каким стоит откликнуться сегодня."
            href="/applicant/saved"
          />
          <QueueCard
            title="Приватность"
            subtitle="Убедись, что профиль, резюме и отклики открыты так, как тебе комфортно."
            href="/applicant/privacy"
          />
        </div>
      </section>

      <section className="section-block">
        <SectionHeading
          eyebrow="Последние отклики"
          title="Текущая воронка"
          action={
            <Link href="/applicant/applications" className="button button--ghost">
              Вся история
            </Link>
          }
        />
        <div className="list-stack">
          {applications.items.slice(0, 3).map((item) => (
            <ApplicationRow key={item.id} item={item} />
          ))}
        </div>
      </section>
    </>
  );
}
