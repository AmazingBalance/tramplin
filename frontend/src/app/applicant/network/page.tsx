import {
  type Connection,
  type ListResponse,
} from "@/lib/api";
import { serverFetchJson } from "@/lib/server-api";

import { EmptyState, QueueCard, SectionHeading, TagStrip } from "@/components/ui";

export default async function ApplicantNetworkPage() {
  const [connections, sentRecommendations, receivedRecommendations] = await Promise.all([
    serverFetchJson<ListResponse<Connection>>("/me/connections?page=1&pageSize=20"),
    serverFetchJson<ListResponse<{ id: string; message: string; opportunityId: string; recipientUserId: string }>>(
      "/me/recommendations/sent?page=1&pageSize=20",
    ),
    serverFetchJson<ListResponse<{ id: string; message: string; opportunityId: string; recommenderUserId: string }>>(
      "/me/recommendations/received?page=1&pageSize=20",
    ),
  ]);

  return (
    <div className="stack-page">
      <section className="section-block">
        <SectionHeading
          eyebrow="Профессиональные контакты"
          title="Сеть, которая помогает не только искать, но и рекомендовать"
          text="По заданию это отдельный продуктовый слой, поэтому он вынесен в самостоятельную рабочую страницу."
        />
        {connections.items.length > 0 ? (
          <div className="queue-grid">
            {connections.items.map((item) => (
              <QueueCard
                key={item.id}
                title={item.otherApplicant.displayName}
                subtitle={`${item.otherApplicant.universityName ?? "Университет не указан"} · ${
                  item.otherApplicant.city ?? "Город не указан"
                }`}
                meta={<TagStrip tags={item.otherApplicant.tags} />}
              />
            ))}
          </div>
        ) : (
          <EmptyState
            title="Контактов пока нет"
            text="Когда начнёшь добавлять связи, здесь появится твоя рабочая сеть."
          />
        )}
      </section>

      <section className="section-block section-block--soft">
        <SectionHeading
          eyebrow="Рекомендации"
          title="Тёплые сигналы от людей, которым ты доверяешь"
        />
        <div className="queue-grid">
          <QueueCard
            title={`Отправлено: ${sentRecommendations.items.length}`}
            subtitle="Сколько раз ты подсказал кому-то подходящую возможность."
          />
          <QueueCard
            title={`Получено: ${receivedRecommendations.items.length}`}
            subtitle="Сколько раз кто-то порекомендовал тебе подходящий шанс."
          />
        </div>
      </section>
    </div>
  );
}
