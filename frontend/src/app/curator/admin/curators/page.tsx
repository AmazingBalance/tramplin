import {
  type CuratorAccount,
  type ListResponse,
} from "@/lib/api";
import { getCurrentUserSafe, serverFetchJsonSafe } from "@/lib/server-api";

import { CuratorCreateForm } from "@/components/workspace-forms";
import { QueueCard, SectionHeading } from "@/components/ui";

export default async function CuratorAdminPage() {
  const [currentUser, curators] = await Promise.all([
    getCurrentUserSafe(),
    serverFetchJsonSafe<ListResponse<CuratorAccount>>("/curator/admin/curators?page=1&pageSize=20"),
  ]);

  return (
    <div className="two-column-page">
      <section className="section-block">
        <SectionHeading
          eyebrow="Команда кураторов"
          title="Admin-layer для расширения команды"
          text="По заданию создавать других кураторов может только admin-curator, поэтому здесь отдельный защищённый экран."
        />
        <div className="queue-grid">
          {(curators?.items ?? []).map((item) => (
            <QueueCard
              key={item.id}
              title={item.displayName}
              subtitle={`${item.email} · ${item.isAdmin ? "admin-curator" : "curator"}`}
            />
          ))}
        </div>
      </section>

      <aside className="side-stack">
        <section className="detail-panel">
          <h2>Новый куратор</h2>
          <CuratorCreateForm isAdmin={Boolean(currentUser?.curatorProfile?.isAdmin)} />
        </section>
      </aside>
    </div>
  );
}
