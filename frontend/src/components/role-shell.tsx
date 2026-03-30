import Image from "next/image";
import Link from "next/link";
import type { ReactNode } from "react";

import { roleNavigation } from "@/lib/content";
import type { CurrentUser, UserRole } from "@/lib/api";
import { roleHomePath } from "@/lib/api";

import { LogoutButton } from "@/components/session-actions";

export function RoleShell({
  role,
  user,
  title,
  description,
  children,
  nudge,
}: {
  role: UserRole;
  user: CurrentUser;
  title: string;
  description: string;
  children: ReactNode;
  nudge?: ReactNode;
}) {
  const nav = roleNavigation[role];

  return (
    <div className="workspace">
      <aside className="workspace__sidebar">
        <Link href={roleHomePath(role)} className="workspace__brand">
          <Image src="/brand/tramplin-mark.svg" alt="" width={42} height={42} aria-hidden />
          TRAMPLIN
        </Link>
        <p className="workspace__brand-copy">Поиск траектории в одной платформе.</p>
        <nav className="workspace__nav">
          {nav.map((item) => (
            <Link key={item.href} href={item.href}>
              {item.label}
            </Link>
          ))}
        </nav>
        <div className="workspace__side-footer">
          <p>{user.displayName}</p>
          <span>{user.email}</span>
          <div className="workspace__side-actions">
            <Link href={roleHomePath(role)} className="button button--ghost">
              Главная роли
            </Link>
            <LogoutButton />
          </div>
        </div>
      </aside>
      <div className="workspace__content">
        <header className="workspace__header">
          <div>
            <p className="eyebrow">
              {role === "applicant"
                ? "Соискатель"
                : role === "employer"
                  ? "Работодатель"
                  : "Куратор"}
            </p>
            <h1>{title}</h1>
            <p>{description}</p>
          </div>
        </header>
        {nudge ? <div className="workspace-nudge">{nudge}</div> : null}
        <main className="workspace__main">{children}</main>
      </div>
    </div>
  );
}
