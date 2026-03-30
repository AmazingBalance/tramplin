import {
  type Application,
  type ListResponse,
} from "@/lib/api";
import { serverFetchJson } from "@/lib/server-api";

import { ApplicationRow, EmptyState, SectionHeading } from "@/components/ui";

export default async function ApplicantApplicationsPage() {
  const applications = await serverFetchJson<ListResponse<Application>>(
    "/me/applications?page=1&pageSize=20",
  );

  return (
    <section className="section-block">
      <SectionHeading
        eyebrow="История откликов"
        title="Вся лента статусов и решений"
        text="Если команда перевела отклик в review, reserve или reject, это видно здесь без лишней путаницы."
      />
      {applications.items.length > 0 ? (
        <div className="list-stack">
          {applications.items.map((item) => (
            <ApplicationRow key={item.id} item={item} />
          ))}
        </div>
      ) : (
        <EmptyState
          title="Откликов ещё нет"
          text="Когда отправишь первый отклик, здесь появится понятная история движения по статусам."
        />
      )}
    </section>
  );
}
