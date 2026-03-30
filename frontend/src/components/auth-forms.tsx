"use client";

import Link from "next/link";
import { useRouter } from "next/navigation";
import { useState } from "react";

import { clientFetchJson, roleHomePath, type CurrentUser, type UserRole } from "@/lib/api";

import { FormNotice, type Notice, prettyError } from "@/components/form-kit";

export function LoginForm() {
  const router = useRouter();
  const [email, setEmail] = useState("");
  const [password, setPassword] = useState("");
  const [pending, setPending] = useState(false);
  const [notice, setNotice] = useState<Notice>(null);

  async function onSubmit(event: React.FormEvent<HTMLFormElement>) {
    event.preventDefault();
    setPending(true);
    setNotice(null);

    try {
      const response = await clientFetchJson<{ user: CurrentUser }>("/auth/login", {
        method: "POST",
        body: JSON.stringify({ email, password }),
      });

      router.push(roleHomePath(response.user.role));
      router.refresh();
    } catch (error) {
      setNotice({ tone: "error", text: prettyError(error) });
    } finally {
      setPending(false);
    }
  }

  return (
    <form className="stack-form" onSubmit={onSubmit}>
      <FormNotice notice={notice} />
      <label>
        <span>Email</span>
        <input
          type="email"
          autoComplete="email"
          value={email}
          onChange={(event) => setEmail(event.target.value)}
          required
        />
      </label>
      <label>
        <span>Пароль</span>
        <input
          type="password"
          autoComplete="current-password"
          value={password}
          onChange={(event) => setPassword(event.target.value)}
          required
        />
      </label>
      <button type="submit" className="button button--primary" disabled={pending}>
        {pending ? "Входим..." : "Войти в кабинет"}
      </button>
    </form>
  );
}

export function RegisterForm() {
  const router = useRouter();
  const [role, setRole] = useState<UserRole>("applicant");
  const [email, setEmail] = useState("");
  const [password, setPassword] = useState("");
  const [confirmPassword, setConfirmPassword] = useState("");
  const [displayName, setDisplayName] = useState("");
  const [firstName, setFirstName] = useState("");
  const [lastName, setLastName] = useState("");
  const [fullName, setFullName] = useState("");
  const [pending, setPending] = useState(false);
  const [notice, setNotice] = useState<Notice>(null);

  async function onSubmit(event: React.FormEvent<HTMLFormElement>) {
    event.preventDefault();

    if (password !== confirmPassword) {
      setNotice({ tone: "error", text: "Пароли не совпадают. Проверь оба поля." });
      return;
    }

    setPending(true);
    setNotice(null);

    const path =
      role === "applicant" ? "/auth/register/applicant" : "/auth/register/employer";
    const payload =
      role === "applicant"
        ? { email, password, displayName, firstName, lastName }
        : { email, password, displayName, fullName };

    try {
      await clientFetchJson(path, {
        method: "POST",
        body: JSON.stringify(payload),
      });
      router.push(role === "applicant" ? "/applicant/profile" : "/employer/company");
      router.refresh();
    } catch (error) {
      setNotice({ tone: "error", text: prettyError(error) });
    } finally {
      setPending(false);
    }
  }

  return (
    <form className="stack-form" onSubmit={onSubmit}>
      <div className="role-switch">
        <button
          type="button"
          className={role === "applicant" ? "role-switch__item is-active" : "role-switch__item"}
          onClick={() => setRole("applicant")}
        >
          Я студент
        </button>
        <button
          type="button"
          className={role === "employer" ? "role-switch__item is-active" : "role-switch__item"}
          onClick={() => setRole("employer")}
        >
          Я работодатель
        </button>
      </div>

      <FormNotice notice={notice} />

      <label>
        <span>Email</span>
        <input
          type="email"
          autoComplete="email"
          value={email}
          onChange={(event) => setEmail(event.target.value)}
          required
        />
      </label>

      <label>
        <span>Имя профиля</span>
        <input
          value={displayName}
          onChange={(event) => setDisplayName(event.target.value)}
          placeholder="Как тебя будет видно в системе"
          required
        />
      </label>

      {role === "applicant" ? (
        <div className="split-fields">
          <label>
            <span>Имя</span>
            <input value={firstName} onChange={(event) => setFirstName(event.target.value)} required />
          </label>
          <label>
            <span>Фамилия</span>
            <input value={lastName} onChange={(event) => setLastName(event.target.value)} required />
          </label>
        </div>
      ) : (
        <label>
          <span>Полное имя контактного лица</span>
          <input value={fullName} onChange={(event) => setFullName(event.target.value)} required />
        </label>
      )}

      <label>
        <span>Пароль</span>
        <input
          type="password"
          autoComplete="new-password"
          value={password}
          onChange={(event) => setPassword(event.target.value)}
          minLength={8}
          required
        />
      </label>

      <label>
        <span>Повторите пароль</span>
        <input
          type="password"
          autoComplete="new-password"
          value={confirmPassword}
          onChange={(event) => setConfirmPassword(event.target.value)}
          minLength={8}
          required
        />
      </label>

      <button type="submit" className="button button--primary" disabled={pending}>
        {pending ? "Создаём аккаунт..." : "Начать через регистрацию"}
      </button>
    </form>
  );
}

export function InlineSeedHint() {
  return (
    <div className="seed-hint">
      <p>Нужен быстрый вход для проверки?</p>
      <Link href="/auth/login">Открыть форму логина</Link>
    </div>
  );
}
