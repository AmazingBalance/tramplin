import {
  type ListResponse,
  type VerificationRequest,
} from "@/lib/api";
import { serverFetchJson } from "@/lib/server-api";

import { EmptyState, SectionHeading, VerificationRow } from "@/components/ui";

export default async function CuratorVerificationPage() {
  const requests = await serverFetchJson<ListResponse<VerificationRequest>>(
    "/curator/verification-requests?page=1&pageSize=20",
  );

  return (
    <section className="section-block">
      <SectionHeading
        eyebrow="Проверки"
        title="Очередь запросов на верификацию компаний"
        text="Куратор должен быстро видеть метод проверки, статус и историю review без переключения контекста."
      />
      {requests.items.length > 0 ? (
        <div className="list-stack">
          {requests.items.map((item) => (
            <VerificationRow key={item.id} item={item} />
          ))}
        </div>
      ) : (
        <EmptyState
          title="Очередь пуста"
          text="Сейчас новых запросов на проверку нет."
        />
      )}
    </section>
  );
}
