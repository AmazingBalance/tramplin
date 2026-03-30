"use client";

import { useRouter } from "next/navigation";
import { useState } from "react";

import { clientFetchJson } from "@/lib/api";

export function LogoutButton() {
  const router = useRouter();
  const [pending, setPending] = useState(false);

  return (
    <button
      type="button"
      className="button button--ghost"
      disabled={pending}
      onClick={async () => {
        setPending(true);
        try {
          await clientFetchJson("/auth/logout", { method: "POST" });
          router.push("/");
          router.refresh();
        } finally {
          setPending(false);
        }
      }}
    >
      {pending ? "Выходим..." : "Выйти"}
    </button>
  );
}
