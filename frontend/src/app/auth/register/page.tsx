import { redirect } from "next/navigation";

import { roleHomePath } from "@/lib/api";
import { getCurrentUserSafe } from "@/lib/server-api";

import { RegisterForm } from "@/components/auth-forms";
import { SiteHeader } from "@/components/site-header";

export default async function RegisterPage() {
  const user = await getCurrentUserSafe();

  if (user) {
    redirect(roleHomePath(user.role));
  }

  return (
    <div className="page-shell">
      <SiteHeader user={user} />
      <main className="auth-layout">
        <section className="auth-copy">
          <p className="eyebrow">Регистрация</p>
          <h1>Сначала выбираешь роль, потом делаешь короткий первый шаг, а не тонешь в длинной анкете.</h1>
          <p>
            Для студента это вход в личный маршрут, для работодателя старт настройки компании и
            верификации.
          </p>
          <div className="auth-copy__notes">
            <div className="seed-card">
              <strong>Что будет дальше</strong>
              <span>После регистрации откроется кабинет с поиском, профилем и следующими шагами.</span>
            </div>
          </div>
        </section>
        <section className="auth-panel">
          <RegisterForm />
        </section>
      </main>
    </div>
  );
}
