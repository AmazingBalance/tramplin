import {
  type Application,
  type ListResponse,
  type Opportunity,
} from "@/lib/api";
import { serverFetchJson } from "@/lib/server-api";

import { EmptyState, QueueCard, SectionHeading } from "@/components/ui";

export default async function EmployerApplicantsPage() {
  const opportunities = await serverFetchJson<ListResponse<Opportunity>>(
    "/employer/opportunities?page=1&pageSize=20",
  );

  const applicationBatches = await Promise.all(
    opportunities.items.map(async (item) => ({
      opportunity: item,
      applications: await serverFetchJson<ListResponse<Application>>(
        `/employer/opportunities/${item.id}/applications?page=1&pageSize=20`,
      ),
    })),
  );

  const totalApplications = applicationBatches.reduce(
    (accumulator, batch) => accumulator + batch.applications.items.length,
    0,
  );

  return (
    <section className="section-block">
      <SectionHeading
        eyebrow="Кандидаты"
        title="Отклики по всем карточкам"
        text="В одном месте видно, кто откликнулся, по какой позиции и в каком статусе сейчас находится."
      />
      {totalApplications > 0 ? (
        <div className="queue-grid">
          {applicationBatches.flatMap((batch) =>
            batch.applications.items.map((item) => (
              <QueueCard
                key={item.id}
                title={item.applicant?.displayName ?? "Кандидат"}
                subtitle={`${batch.opportunity.title} · ${item.status}`}
              />
            )),
          )}
        </div>
      ) : (
        <EmptyState
          title="Кандидатов пока нет"
          text="После первых откликов здесь появится операционная очередь по каждой карточке."
        />
      )}
    </section>
  );
}
