import { redirect } from "next/navigation";

import { roleHomePath } from "@/lib/api";
import { getCurrentUserSafe } from "@/lib/server-api";

import { LoginForm } from "@/components/auth-forms";
import { SiteHeader } from "@/components/site-header";

export default async function LoginPage() {
  const user = await getCurrentUserSafe();

  if (user) {
    redirect(roleHomePath(user.role));
  }

  return (
    <div className="page-shell">
      <SiteHeader user={user} />
      <main className="auth-layout">
        <section className="auth-copy">
          <p className="eyebrow">Вход</p>
          <h1>Быстрый возврат в рабочий сценарий без маркетингового шума.</h1>
          <p>
            Логин открывает личный кабинет соискателя, работодателя или куратора и возвращает в
            рабочий сценарий без лишних шагов.
          </p>
          <div className="auth-copy__notes">
            <div className="seed-card">
              <strong>После входа</strong>
              <span>Система сразу переведёт в кабинет вашей роли и сохранит рабочий контекст.</span>
            </div>
          </div>
        </section>
        <section className="auth-panel">
          <LoginForm />
        </section>
      </main>
    </div>
  );
}
