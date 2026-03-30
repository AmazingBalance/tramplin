"use client";

import type { ReactNode } from "react";

import { ApiError } from "@/lib/api";

export type Notice = {
  tone: "error" | "success";
  text: string;
} | null;

export function prettyError(error: unknown) {
  if (error instanceof ApiError) {
    return error.message;
  }
  return "Что-то пошло не так. Попробуй ещё раз.";
}

export function FormNotice({ notice }: { notice: Notice }) {
  if (!notice) {
    return null;
  }

  return (
    <div className={`form-notice form-notice--${notice.tone}`}>{notice.text}</div>
  );
}

export function FormCard({
  title,
  text,
  children,
}: {
  title: string;
  text?: string;
  children: ReactNode;
}) {
  return (
    <section className="form-card">
      <div className="form-card__header">
        <h3>{title}</h3>
        {text ? <p>{text}</p> : null}
      </div>
      {children}
    </section>
  );
}
