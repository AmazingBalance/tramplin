import Image from "next/image";
import Link from "next/link";

import type { CurrentUser } from "@/lib/api";
import { roleHomePath } from "@/lib/api";

import { LogoutButton } from "@/components/session-actions";

export function SiteHeader({ user }: { user: CurrentUser | null }) {
  const brandHref = user ? roleHomePath(user.role) : "/";

  return (
    <header className="site-header">
      <div className="site-header__inner">
        <Link href={brandHref} className="site-header__brand">
          <Image src="/brand/tramplin-mark.svg" alt="" width={48} height={48} aria-hidden />
          <span>TRAMPLIN</span>
        </Link>
        {!user ? (
          <nav className="site-header__nav">
            <Link href="/#strengths">Сильные стороны</Link>
            <Link href="/#teams">Для компаний</Link>
            <Link href="/#mission">Миссия</Link>
          </nav>
        ) : (
          <nav className="site-header__nav">
            <Link href={roleHomePath(user.role)}>Главная роли</Link>
          </nav>
        )}
        <div className="site-header__actions">
          {user ? (
            <>
              <Link href={roleHomePath(user.role)} className="button button--ghost">
                Кабинет
              </Link>
              <LogoutButton />
            </>
          ) : (
            <>
              <Link href="/auth/login" className="button button--ghost">
                Войти
              </Link>
              <Link href="/auth/register" className="button button--primary">
                Регистрация
              </Link>
            </>
          )}
        </div>
      </div>
    </header>
  );
}
