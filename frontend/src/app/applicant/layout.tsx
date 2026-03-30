import Link from "next/link";

import { getCurrentUserSafe } from "@/lib/server-api";

import { RoleShell } from "@/components/role-shell";
import { SiteHeader } from "@/components/site-header";
import { GuardCard } from "@/components/ui";

export default async function ApplicantLayout({
  children,
}: {
  children: React.ReactNode;
}) {
  const user = await getCurrentUserSafe();

  if (!user || user.role !== "applicant") {
    return (
      <div className="page-shell">
        <SiteHeader user={user} />
        <main className="guard-wrap">
          <GuardCard
            title="Кабинет соискателя доступен после входа"
            text="Зайди как студент или создай аккаунт, чтобы открыть профиль, отклики, сохранённое и нетворкинг."
            href="/auth/register"
            actionLabel="Перейти к регистрации"
          />
        </main>
      </div>
    );
  }

  return (
    <RoleShell
      role="applicant"
      user={user}
      title="Кабинет соискателя"
      description="Здесь собираются профиль, отклики, сохранённое, сеть контактов и настройки приватности."
      nudge={
        <div className="nudge-card">
          <div>
            <strong>Профиль почти готов</strong>
            <p>Добавь ещё пару блоков, и рекомендации станут точнее.</p>
          </div>
          <Link href="/applicant/profile" className="button button--primary">
            Дополнить профиль
          </Link>
        </div>
      }
    >
      {children}
    </RoleShell>
  );
}
