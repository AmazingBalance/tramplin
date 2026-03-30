"use client";

import { useRouter } from "next/navigation";
import { useEffect, useState } from "react";

import { clientFetchJson, type ListResponse, type UserRole } from "@/lib/api";

import { FormNotice, type Notice, prettyError } from "@/components/form-kit";

function readGuestSaved(key: string) {
  if (typeof window === "undefined") {
    return [] as string[];
  }

  try {
    const raw = window.localStorage.getItem(key);
    return raw ? (JSON.parse(raw) as string[]) : [];
  } catch {
    return [];
  }
}

function writeGuestSaved(key: string, ids: string[]) {
  if (typeof window !== "undefined") {
    window.localStorage.setItem(key, JSON.stringify(ids));
  }
}

export function SaveOpportunityButton({
  opportunityId,
  currentRole,
}: {
  opportunityId: string;
  currentRole: UserRole | null;
}) {
  const [saved, setSaved] = useState(false);
  const [pending, setPending] = useState(false);
  const [notice, setNotice] = useState<Notice>(null);
  const storage = "tramplin_guest_opportunities";

  useEffect(() => {
    let active = true;

    async function sync() {
      if (currentRole === "applicant") {
        try {
          const response = await clientFetchJson<ListResponse<{ opportunityId: string }>>(
            "/me/saved-opportunities?page=1&pageSize=100",
          );
          if (active) {
            setSaved(response.items.some((item) => item.opportunityId === opportunityId));
          }
          return;
        } catch {
          // fallback
        }
      }

      if (active) {
        setSaved(readGuestSaved(storage).includes(opportunityId));
      }
    }

    sync();

    return () => {
      active = false;
    };
  }, [currentRole, opportunityId]);

  async function toggle() {
    setPending(true);
    setNotice(null);
    try {
      if (currentRole === "applicant") {
        if (saved) {
          await clientFetchJson(`/me/saved-opportunities/${opportunityId}`, {
            method: "DELETE",
          });
        } else {
          await clientFetchJson("/me/saved-opportunities", {
            method: "POST",
            body: JSON.stringify({ opportunityId }),
          });
        }
      } else {
        const ids = new Set(readGuestSaved(storage));
        if (ids.has(opportunityId)) {
          ids.delete(opportunityId);
        } else {
          ids.add(opportunityId);
        }
        writeGuestSaved(storage, Array.from(ids));
      }

      setSaved((current) => !current);
    } catch (error) {
      setNotice({ tone: "error", text: prettyError(error) });
    } finally {
      setPending(false);
    }
  }

  return (
    <div className="inline-action">
      <button type="button" className="button button--ghost" onClick={toggle} disabled={pending}>
        {saved ? "Убрано из сохранённого" : "Сохранить"}
      </button>
      <FormNotice notice={notice} />
    </div>
  );
}

export function SaveCompanyButton({
  companyId,
  currentRole,
}: {
  companyId: string;
  currentRole: UserRole | null;
}) {
  const [saved, setSaved] = useState(false);
  const [pending, setPending] = useState(false);
  const storage = "tramplin_guest_companies";

  useEffect(() => {
    let active = true;

    async function sync() {
      if (currentRole === "applicant") {
        try {
          const response = await clientFetchJson<ListResponse<{ companyId: string }>>(
            "/me/saved-companies?page=1&pageSize=100",
          );
          if (active) {
            setSaved(response.items.some((item) => item.companyId === companyId));
          }
          return;
        } catch {
          // fallback
        }
      }

      if (active) {
        setSaved(readGuestSaved(storage).includes(companyId));
      }
    }

    sync();

    return () => {
      active = false;
    };
  }, [companyId, currentRole]);

  async function toggle() {
    setPending(true);
    try {
      if (currentRole === "applicant") {
        if (saved) {
          await clientFetchJson(`/me/saved-companies/${companyId}`, {
            method: "DELETE",
          });
        } else {
          await clientFetchJson("/me/saved-companies", {
            method: "POST",
            body: JSON.stringify({ companyId }),
          });
        }
      } else {
        const ids = new Set(readGuestSaved(storage));
        if (ids.has(companyId)) {
          ids.delete(companyId);
        } else {
          ids.add(companyId);
        }
        writeGuestSaved(storage, Array.from(ids));
      }

      setSaved((current) => !current);
    } finally {
      setPending(false);
    }
  }

  return (
    <button type="button" className="button button--ghost" onClick={toggle} disabled={pending}>
      {saved ? "Сохранено" : "Сохранить"}
    </button>
  );
}

export function ApplicationComposer({
  opportunityId,
  currentRole,
}: {
  opportunityId: string;
  currentRole: UserRole | null;
}) {
  const router = useRouter();
  const [coverLetter, setCoverLetter] = useState("");
  const [pending, setPending] = useState(false);
  const [notice, setNotice] = useState<Notice>(null);

  async function onSubmit(event: React.FormEvent<HTMLFormElement>) {
    event.preventDefault();

    if (currentRole !== "applicant") {
      router.push("/auth/login");
      return;
    }

    setPending(true);
    setNotice(null);

    try {
      await clientFetchJson("/me/applications", {
        method: "POST",
        body: JSON.stringify({
          opportunityId,
          coverLetter: coverLetter.trim() || undefined,
        }),
      });
      setNotice({ tone: "success", text: "Отклик отправлен. Статус появится в кабинете." });
      setCoverLetter("");
      router.refresh();
    } catch (error) {
      setNotice({ tone: "error", text: prettyError(error) });
    } finally {
      setPending(false);
    }
  }

  return (
    <form className="stack-form stack-form--compact" onSubmit={onSubmit}>
      <label>
        <span>Короткое сообщение команде</span>
        <textarea
          value={coverLetter}
          onChange={(event) => setCoverLetter(event.target.value)}
          rows={4}
          placeholder="Пара строк о мотивации, опыте или интересе к продукту."
        />
      </label>
      <button type="submit" className="button button--primary" disabled={pending}>
        {currentRole === "applicant"
          ? pending
            ? "Отправляем..."
            : "Откликнуться"
          : "Войти как соискатель"}
      </button>
      <FormNotice notice={notice} />
    </form>
  );
}
