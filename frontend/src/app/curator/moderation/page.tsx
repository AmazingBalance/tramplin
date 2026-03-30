import {
  type ListResponse,
  type ModerationCase,
} from "@/lib/api";
import { serverFetchJson } from "@/lib/server-api";

import { EmptyState, ModerationRow, SectionHeading } from "@/components/ui";

export default async function CuratorModerationPage() {
  const moderationCases = await serverFetchJson<ListResponse<ModerationCase>>(
    "/curator/moderation-cases?page=1&pageSize=20",
  );

  return (
    <section className="section-block">
      <SectionHeading
        eyebrow="Модерация"
        title="Кейсы, где нужно решение по карточке, компании или профилю"
        text="Это отдельная операционная очередь, а не просто копия ленты проверок."
      />
      {moderationCases.items.length > 0 ? (
        <div className="list-stack">
          {moderationCases.items.map((item) => (
            <ModerationRow key={item.id} item={item} />
          ))}
        </div>
      ) : (
        <EmptyState title="Кейсов нет" text="Новых moderation-case сейчас нет." />
      )}
    </section>
  );
}
