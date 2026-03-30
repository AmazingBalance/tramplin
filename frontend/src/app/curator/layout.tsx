import { getCurrentUserSafe } from "@/lib/server-api";

import { RoleShell } from "@/components/role-shell";
import { SiteHeader } from "@/components/site-header";
import { GuardCard } from "@/components/ui";

export default async function CuratorLayout({
  children,
}: {
  children: React.ReactNode;
}) {
  const user = await getCurrentUserSafe();

  if (!user || user.role !== "curator") {
    return (
      <div className="page-shell">
        <SiteHeader user={user} />
        <main className="guard-wrap">
          <GuardCard
            title="Кураторский контур доступен только кураторам"
            text="Вход по роли куратора нужен для управления проверками, модерацией и правками карточек."
            href="/auth/login"
            actionLabel="Открыть логин"
          />
        </main>
      </div>
    );
  }

  return (
    <RoleShell
      role="curator"
      user={user}
      title="Кураторский контур"
      description="Рабочий интерфейс для проверок, модерации, исправлений и управления командой кураторов."
    >
      {children}
    </RoleShell>
  );
}
