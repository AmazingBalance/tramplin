import {
  type ApplicantPrivacy,
} from "@/lib/api";
import { serverFetchJson } from "@/lib/server-api";

import { ApplicantPrivacyForm } from "@/components/workspace-forms";
import { SectionHeading } from "@/components/ui";

export default async function ApplicantPrivacyPage() {
  const privacy = await serverFetchJson<ApplicantPrivacy>("/me/applicant/privacy");

  return (
    <div className="two-column-page">
      <section className="section-block">
        <SectionHeading
          eyebrow="Приватность"
          title="Ты сам решаешь, что видно другим"
          text="Отдельный экран приватности помогает не смешивать чувствительные настройки с редактированием профиля."
        />
        <ApplicantPrivacyForm privacy={privacy} />
      </section>

      <aside className="side-stack">
        <section className="detail-panel">
          <h2>Как это работает</h2>
          <p>Профиль можно оставить только для контактов, резюме скрыть, а карьерные интересы наоборот показать.</p>
        </section>
      </aside>
    </div>
  );
}
