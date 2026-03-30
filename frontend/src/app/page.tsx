import Image from "next/image";
import Link from "next/link";

import {
    composeLocation,
    formatOpportunityType,
    formatParticipation,
    roleHomePath,
    type ListResponse,
    type Opportunity,
} from "@/lib/api";
import {
    heroHighlights,
    homeStats,
    landingFeatureStories,
    missionPoints,
} from "@/lib/content";
import { getCurrentUserSafe, serverFetchJson } from "@/lib/server-api";

import { LandingFeatureStories } from "@/components/landing-feature-stories";
import { SiteHeader } from "@/components/site-header";
import { MetricCard, SectionHeading } from "@/components/ui";

const fallbackFeatured = [
    {
        id: "internship-preview",
        title: "Backend Internship",
        company: { brandName: "Data River" },
        type: "internship",
        participationFormat: "hybrid",
        locationLabel: "Москва",
    },
    {
        id: "event-preview",
        title: "Ночной хакатон",
        company: { brandName: "Signal Lab" },
        type: "event",
        participationFormat: "offline",
        locationLabel: "Казань",
    },
    {
        id: "mentor-preview",
        title: "Трек с ментором",
        company: { brandName: "North Star" },
        type: "mentor_program",
        participationFormat: "remote",
        locationLabel: "Онлайн",
    },
];

export default async function HomePage() {
    const user = await getCurrentUserSafe();

    const opportunities = await serverFetchJson<ListResponse<Opportunity>>(
        "/public/opportunities?page=1&pageSize=6",
    );

    const featured =
        opportunities.items.length > 0
            ? opportunities.items.slice(0, 3).map((item) => ({
                  id: item.id,
                  title: item.title,
                  company: { brandName: item.company.brandName },
                  type: item.type,
                  participationFormat: item.participationFormat,
                  locationLabel: composeLocation(item.location),
              }))
            : fallbackFeatured;
    const primaryHref = user ? roleHomePath(user.role) : "/auth/register";
    const primaryLabel = user ? "Открыть кабинет" : "Создать профиль";

    return (
        <div className="page-shell">
            <SiteHeader user={user} />
            <main className="landing">
                <section className="hero hero--landing">
                    <div className="hero__copy">
                        <p className="eyebrow">TRAMPLIN</p>
                        <h1>Найди свой старт.</h1>
                        <p className="hero__lead">
                            Стажировки, junior-вакансии, менторские программы и
                            карьерные события в одной платформе.
                        </p>
                        <div className="hero__cta-wrap">
                            <Link
                                href={primaryHref}
                                className="button button--primary button--hero"
                            >
                                {primaryLabel}
                            </Link>
                            <div className="hero__signal">
                                <strong>12 400+</strong>
                                <span>
                                    студентов и молодых специалистов уже на
                                    платформе
                                </span>
                            </div>
                        </div>
                        <div className="hero__chips">
                            {heroHighlights.map((item) => (
                                <div key={item.title} className="hero-chip">
                                    <strong>{item.title}</strong>
                                    <span>{item.text}</span>
                                </div>
                            ))}
                        </div>
                    </div>

                    <div className="hero__cards">
                        <Image
                            src="/visuals/path-field.svg"
                            alt=""
                            fill
                            aria-hidden
                            className="hero__cards-art"
                            sizes="(max-width: 1120px) 100vw, 520px"
                        />
                        <div className="hero__cards-stack">
                            {featured.map((item) => (
                                <article key={item.id} className="hero-preview">
                                    <div className="hero-preview__top">
                                        <span className="hero-preview__type">
                                            {formatOpportunityType(item.type)}
                                        </span>
                                        <span className="hero-preview__format">
                                            {formatParticipation(
                                                item.participationFormat,
                                            )}
                                        </span>
                                    </div>
                                    <div className="hero-preview__body">
                                        <strong>{item.title}</strong>
                                        <p>{item.company.brandName}</p>
                                        <span>{item.locationLabel}</span>
                                    </div>
                                </article>
                            ))}
                        </div>
                        <div className="hero__visual-note">
                            <Image
                                src="/visuals/trail-orbit.svg"
                                alt=""
                                width={144}
                                height={112}
                                aria-hidden
                                className="hero__visual-orbit"
                            />
                            <div>
                                <strong>Карта и лента уже внутри</strong>
                                <span>
                                    После регистрации главная страница
                                    собирается под роль пользователя.
                                </span>
                            </div>
                        </div>
                    </div>
                </section>

                <section className="metrics-strip">
                    {homeStats.map((item, index) => (
                        <MetricCard
                            key={item.label}
                            label={item.label}
                            value={item.value}
                            tone={index === 0 ? "accent" : "soft"}
                        />
                    ))}
                </section>

                <section
                    id="strengths"
                    className="section-block section-block--glass"
                >
                    <SectionHeading
                        eyebrow="Сильные стороны"
                        title="Смотри коротко, открывай глубже."
                        text="На витрине только визуальные сценарии. Подробности о каждом процессе раскрываются по клику."
                    />
                    <LandingFeatureStories items={landingFeatureStories} />
                </section>

                <section id="teams" className="section-cta">
                    <div>
                        <p className="eyebrow">Для компаний</p>
                        <h2>
                            Привлекайте кандидатов и собирайте участников на
                            карьерные события в одном контуре.
                        </h2>
                        <p>
                            Компания может публиковать вакансии, стажировки,
                            менторские программы, лекции, хакатоны и другие
                            карьерные мероприятия без разрыва между каналами.
                        </p>
                    </div>
                    <div className="section-cta__actions">
                        <Image
                            src="/visuals/trail-orbit.svg"
                            alt=""
                            width={220}
                            height={170}
                            aria-hidden
                            className="section-cta__visual"
                        />
                        <Link
                            href={primaryHref}
                            className="button button--primary"
                        >
                            {user ? "Перейти в кабинет" : "Подключить компанию"}
                        </Link>
                    </div>
                </section>

                <section
                    id="mission"
                    className="section-block section-block--quiet"
                >
                    <SectionHeading
                        eyebrow="Миссия"
                        title="Помочь молодым специалистам построить карьеру и тем самым усилить IT-среду."
                        text="Когда вход в профессию становится понятнее, выигрывают и сами люди, и команды, которые ищут сильных молодых участников."
                    />
                    <div className="category-grid">
                        {missionPoints.map((item) => (
                            <article key={item.title} className="category-card">
                                <h3>{item.title}</h3>
                                <p>{item.text}</p>
                            </article>
                        ))}
                    </div>
                </section>
            </main>
        </div>
    );
}
