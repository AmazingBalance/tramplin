import {
  type CurrentUser,
  type ListResponse,
} from "@/lib/api";
import { serverFetchJson } from "@/lib/server-api";

import { EmptyState, QueueCard, SectionHeading } from "@/components/ui";

export default async function CuratorUsersPage() {
  const users = await serverFetchJson<ListResponse<CurrentUser>>(
    "/curator/users?page=1&pageSize=20",
  );

  return (
    <section className="section-block">
      <SectionHeading
        eyebrow="Пользователи"
        title="Лента аккаунтов по ролям"
        text="Отсюда куратор видит applicant и employer аккаунты, их активность и текущий статус."
      />
      {users.items.length > 0 ? (
        <div className="queue-grid">
          {users.items.map((item) => (
            <QueueCard
              key={item.id}
              title={item.displayName}
              subtitle={`${item.role} · ${item.email}`}
            />
          ))}
        </div>
      ) : (
        <EmptyState title="Пользователей нет" text="Очередь пуста." />
      )}
    </section>
  );
}
