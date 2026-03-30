import {
  type ApplicantProfile,
  type ListResponse,
  type SocialLink,
  type Tag,
} from "@/lib/api";
import { serverFetchJson } from "@/lib/server-api";

import { ApplicantProfileForm } from "@/components/workspace-forms";
import { SectionHeading, TagStrip } from "@/components/ui";

export default async function ApplicantProfilePage() {
  const [profile, tags, socialLinks] = await Promise.all([
    serverFetchJson<ApplicantProfile>("/me/applicant-profile"),
    serverFetchJson<ListResponse<Tag>>("/me/applicant/tags"),
    serverFetchJson<ListResponse<SocialLink>>("/me/applicant/social-links"),
  ]);

  return (
    <div className="two-column-page">
      <section className="section-block">
        <SectionHeading
          eyebrow="Профиль"
          title="Собери понятный и живой профиль"
          text="Сейчас здесь базовые поля, которые уже влияют на видимость и рекомендации. Следующим проходом можно докрутить управление ссылками и тегами прямо отсюда."
        />
        <ApplicantProfileForm profile={profile} />
      </section>

      <aside className="side-stack">
        <section className="detail-panel">
          <h2>Текущие теги</h2>
          {tags.items.length > 0 ? <TagStrip tags={tags.items} /> : <p>Теги пока не добавлены.</p>}
        </section>
        <section className="detail-panel">
          <h2>Публичные ссылки</h2>
          {socialLinks.items.length > 0 ? (
            <div className="list-stack">
              {socialLinks.items.map((item) => (
                <div key={item.id} className="mini-company-card">
                  <div>
                    <strong>{item.platform}</strong>
                    <span>{item.url}</span>
                  </div>
                </div>
              ))}
            </div>
          ) : (
            <p>Ссылки пока не добавлены.</p>
          )}
        </section>
      </aside>
    </div>
  );
}
