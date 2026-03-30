import { getCurrentUserSafe } from "@/lib/server-api";

import { RoleShell } from "@/components/role-shell";
import { SiteHeader } from "@/components/site-header";
import { GuardCard } from "@/components/ui";

export default async function EmployerLayout({
  children,
}: {
  children: React.ReactNode;
}) {
  const user = await getCurrentUserSafe();

  if (!user || user.role !== "employer") {
    return (
      <div className="page-shell">
        <SiteHeader user={user} />
        <main className="guard-wrap">
          <GuardCard
            title="Кабинет работодателя доступен после входа"
            text="Зайди как работодатель, чтобы управлять компанией, возможностями, откликами и верификацией."
            href="/auth/register"
            actionLabel="Открыть регистрацию"
          />
        </main>
      </div>
    );
  }

  return (
    <RoleShell
      role="employer"
      user={user}
      title="Кабинет работодателя"
      description="Здесь собираются профиль компании, карточки возможностей, отклики и задачи по верификации."
    >
      {children}
    </RoleShell>
  );
}
