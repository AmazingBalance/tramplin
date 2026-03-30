"use client";

import { useRouter } from "next/navigation";
import { useState } from "react";

import {
  clientFetchJson,
  type ApplicantPrivacy,
  type ApplicantProfile,
  type CompanyMembership,
  type CuratorAccount,
} from "@/lib/api";

import { FormNotice, type Notice, prettyError } from "@/components/form-kit";

export function ApplicantProfileForm({
  profile,
}: {
  profile: ApplicantProfile;
}) {
  const router = useRouter();
  const [form, setForm] = useState({
    firstName: profile.firstName ?? "",
    lastName: profile.lastName ?? "",
    middleName: profile.middleName ?? "",
    universityName: profile.universityName ?? "",
    programName: profile.programName ?? "",
    faculty: profile.faculty ?? "",
    studyYear: profile.studyYear ? String(profile.studyYear) : "",
    graduationYear: profile.graduationYear ? String(profile.graduationYear) : "",
    city: profile.city ?? "",
    about: profile.about ?? "",
  });
  const [pending, setPending] = useState(false);
  const [notice, setNotice] = useState<Notice>(null);

  async function onSubmit(event: React.FormEvent<HTMLFormElement>) {
    event.preventDefault();
    setPending(true);
    setNotice(null);

    try {
      await clientFetchJson("/me/applicant-profile", {
        method: "PATCH",
        body: JSON.stringify({
          firstName: form.firstName,
          lastName: form.lastName,
          middleName: form.middleName || null,
          universityName: form.universityName || null,
          programName: form.programName || null,
          faculty: form.faculty || null,
          studyYear: form.studyYear ? Number(form.studyYear) : null,
          graduationYear: form.graduationYear ? Number(form.graduationYear) : null,
          city: form.city || null,
          about: form.about || null,
        }),
      });
      setNotice({ tone: "success", text: "Профиль обновлён." });
      router.refresh();
    } catch (error) {
      setNotice({ tone: "error", text: prettyError(error) });
    } finally {
      setPending(false);
    }
  }

  return (
    <form className="stack-form" onSubmit={onSubmit}>
      <div className="split-fields">
        <label>
          <span>Имя</span>
          <input
            value={form.firstName}
            onChange={(event) => setForm({ ...form, firstName: event.target.value })}
          />
        </label>
        <label>
          <span>Фамилия</span>
          <input
            value={form.lastName}
            onChange={(event) => setForm({ ...form, lastName: event.target.value })}
          />
        </label>
      </div>
      <div className="split-fields">
        <label>
          <span>Город</span>
          <input value={form.city} onChange={(event) => setForm({ ...form, city: event.target.value })} />
        </label>
        <label>
          <span>Отчество</span>
          <input
            value={form.middleName}
            onChange={(event) => setForm({ ...form, middleName: event.target.value })}
          />
        </label>
      </div>
      <label>
        <span>Университет</span>
        <input
          value={form.universityName}
          onChange={(event) => setForm({ ...form, universityName: event.target.value })}
        />
      </label>
      <div className="split-fields">
        <label>
          <span>Программа</span>
          <input
            value={form.programName}
            onChange={(event) => setForm({ ...form, programName: event.target.value })}
          />
        </label>
        <label>
          <span>Факультет</span>
          <input
            value={form.faculty}
            onChange={(event) => setForm({ ...form, faculty: event.target.value })}
          />
        </label>
      </div>
      <div className="split-fields">
        <label>
          <span>Курс</span>
          <input
            value={form.studyYear}
            onChange={(event) => setForm({ ...form, studyYear: event.target.value })}
          />
        </label>
        <label>
          <span>Год выпуска</span>
          <input
            value={form.graduationYear}
            onChange={(event) => setForm({ ...form, graduationYear: event.target.value })}
          />
        </label>
      </div>
      <label>
        <span>О себе</span>
        <textarea
          rows={5}
          value={form.about}
          onChange={(event) => setForm({ ...form, about: event.target.value })}
        />
      </label>
      <button type="submit" className="button button--primary" disabled={pending}>
        {pending ? "Сохраняем..." : "Сохранить профиль"}
      </button>
      <FormNotice notice={notice} />
    </form>
  );
}

export function ApplicantPrivacyForm({
  privacy,
}: {
  privacy: ApplicantPrivacy;
}) {
  const router = useRouter();
  const [form, setForm] = useState(privacy);
  const [pending, setPending] = useState(false);
  const [notice, setNotice] = useState<Notice>(null);

  async function onSubmit(event: React.FormEvent<HTMLFormElement>) {
    event.preventDefault();
    setPending(true);
    setNotice(null);

    try {
      await clientFetchJson("/me/applicant/privacy", {
        method: "PATCH",
        body: JSON.stringify(form),
      });
      setNotice({ tone: "success", text: "Настройки приватности сохранены." });
      router.refresh();
    } catch (error) {
      setNotice({ tone: "error", text: prettyError(error) });
    } finally {
      setPending(false);
    }
  }

  const options = [
    { value: "private", label: "Только мне" },
    { value: "contacts_only", label: "Только контактам" },
    { value: "authenticated_public", label: "Всем авторизованным" },
  ];

  return (
    <form className="stack-form" onSubmit={onSubmit}>
      <label>
        <span>Видимость профиля</span>
        <select
          value={form.profileVisibility}
          onChange={(event) => setForm({ ...form, profileVisibility: event.target.value })}
        >
          {options.map((item) => (
            <option key={item.value} value={item.value}>
              {item.label}
            </option>
          ))}
        </select>
      </label>
      <label>
        <span>Видимость резюме</span>
        <select
          value={form.resumeVisibility}
          onChange={(event) => setForm({ ...form, resumeVisibility: event.target.value })}
        >
          {options.map((item) => (
            <option key={item.value} value={item.value}>
              {item.label}
            </option>
          ))}
        </select>
      </label>
      <label>
        <span>Видимость откликов</span>
        <select
          value={form.applicationsVisibility}
          onChange={(event) => setForm({ ...form, applicationsVisibility: event.target.value })}
        >
          {options.map((item) => (
            <option key={item.value} value={item.value}>
              {item.label}
            </option>
          ))}
        </select>
      </label>
      <label>
        <span>Видимость контактов</span>
        <select
          value={form.contactsVisibility}
          onChange={(event) => setForm({ ...form, contactsVisibility: event.target.value })}
        >
          {options.map((item) => (
            <option key={item.value} value={item.value}>
              {item.label}
            </option>
          ))}
        </select>
      </label>
      <label className="toggle-field">
        <input
          type="checkbox"
          checked={form.showCareerInterests}
          onChange={(event) =>
            setForm({ ...form, showCareerInterests: event.target.checked })
          }
        />
        <span>Показывать карьерные интересы и направления</span>
      </label>
      <button type="submit" className="button button--primary" disabled={pending}>
        {pending ? "Сохраняем..." : "Сохранить приватность"}
      </button>
      <FormNotice notice={notice} />
    </form>
  );
}

export function OpportunityCreateForm({
  companies,
}: {
  companies: CompanyMembership[];
}) {
  const router = useRouter();
  const [pending, setPending] = useState(false);
  const [notice, setNotice] = useState<Notice>(null);
  const [form, setForm] = useState({
    companyId: companies[0]?.companyId ?? "",
    title: "",
    summary: "",
    slug: "",
    description: "",
    type: "internship",
    participationFormat: "remote",
    city: "",
    country: "Russia",
    addressLine: "",
    employmentType: "full_time",
    experienceLevel: "junior",
    salaryFrom: "",
    salaryTo: "",
    currency: "RUB",
  });

  async function onSubmit(event: React.FormEvent<HTMLFormElement>) {
    event.preventDefault();
    setPending(true);
    setNotice(null);

    try {
      await clientFetchJson("/employer/opportunities", {
        method: "POST",
        body: JSON.stringify({
          companyId: form.companyId,
          title: form.title,
          summary: form.summary,
          slug: form.slug,
          description: form.description,
          type: form.type,
          participationFormat: form.participationFormat,
          location: {
            precision: form.addressLine ? "exact_address" : "city_only",
            country: form.country,
            city: form.city || null,
            addressLine: form.addressLine || null,
          },
          vacancyDetails: {
            employmentType: form.employmentType,
            experienceLevel: form.experienceLevel,
            salaryFrom: form.salaryFrom ? Number(form.salaryFrom) : null,
            salaryTo: form.salaryTo ? Number(form.salaryTo) : null,
            currency: form.currency,
          },
        }),
      });
      router.push("/employer/opportunities");
      router.refresh();
    } catch (error) {
      setNotice({ tone: "error", text: prettyError(error) });
    } finally {
      setPending(false);
    }
  }

  return (
    <form className="stack-form" onSubmit={onSubmit}>
      <label>
        <span>Компания</span>
        <select
          value={form.companyId}
          onChange={(event) => setForm({ ...form, companyId: event.target.value })}
        >
          {companies.map((membership) => (
            <option key={membership.id} value={membership.companyId}>
              {membership.company.brandName}
            </option>
          ))}
        </select>
      </label>
      <label>
        <span>Название</span>
        <input value={form.title} onChange={(event) => setForm({ ...form, title: event.target.value })} required />
      </label>
      <label>
        <span>Короткое описание</span>
        <input value={form.summary} onChange={(event) => setForm({ ...form, summary: event.target.value })} required />
      </label>
      <label>
        <span>Slug</span>
        <input value={form.slug} onChange={(event) => setForm({ ...form, slug: event.target.value })} required />
      </label>
      <label>
        <span>Полное описание</span>
        <textarea
          rows={5}
          value={form.description}
          onChange={(event) => setForm({ ...form, description: event.target.value })}
          required
        />
      </label>
      <div className="split-fields">
        <label>
          <span>Тип</span>
          <select value={form.type} onChange={(event) => setForm({ ...form, type: event.target.value })}>
            <option value="internship">Стажировка</option>
            <option value="vacancy">Вакансия</option>
            <option value="mentor_program">Менторская программа</option>
            <option value="event">Карьерное событие</option>
          </select>
        </label>
        <label>
          <span>Формат</span>
          <select
            value={form.participationFormat}
            onChange={(event) => setForm({ ...form, participationFormat: event.target.value })}
          >
            <option value="remote">Удалённо</option>
            <option value="hybrid">Гибрид</option>
            <option value="offline">Очно</option>
          </select>
        </label>
      </div>
      <div className="split-fields">
        <label>
          <span>Город</span>
          <input value={form.city} onChange={(event) => setForm({ ...form, city: event.target.value })} />
        </label>
        <label>
          <span>Адрес</span>
          <input
            value={form.addressLine}
            onChange={(event) => setForm({ ...form, addressLine: event.target.value })}
          />
        </label>
      </div>
      <div className="split-fields">
        <label>
          <span>Доход от</span>
          <input
            value={form.salaryFrom}
            onChange={(event) => setForm({ ...form, salaryFrom: event.target.value })}
          />
        </label>
        <label>
          <span>Доход до</span>
          <input
            value={form.salaryTo}
            onChange={(event) => setForm({ ...form, salaryTo: event.target.value })}
          />
        </label>
      </div>
      <button type="submit" className="button button--primary" disabled={pending}>
        {pending ? "Создаём..." : "Создать карточку"}
      </button>
      <FormNotice notice={notice} />
    </form>
  );
}

export function CompanyCreateForm() {
  const router = useRouter();
  const [pending, setPending] = useState(false);
  const [notice, setNotice] = useState<Notice>(null);
  const [legalName, setLegalName] = useState("");
  const [slug, setSlug] = useState("");

  async function onSubmit(event: React.FormEvent<HTMLFormElement>) {
    event.preventDefault();
    setPending(true);
    setNotice(null);

    try {
      await clientFetchJson("/employer/companies", {
        method: "POST",
        body: JSON.stringify({
          legalName,
          slug,
        }),
      });
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
        <span>Юридическое название</span>
        <input value={legalName} onChange={(event) => setLegalName(event.target.value)} required />
      </label>
      <label>
        <span>Slug компании</span>
        <input value={slug} onChange={(event) => setSlug(event.target.value)} required />
      </label>
      <button type="submit" className="button button--primary" disabled={pending}>
        {pending ? "Создаём..." : "Создать компанию"}
      </button>
      <FormNotice notice={notice} />
    </form>
  );
}

export function VerificationRequestForm({ companyId }: { companyId: string }) {
  const router = useRouter();
  const [pending, setPending] = useState(false);
  const [notice, setNotice] = useState<Notice>(null);
  const [value, setValue] = useState("https://");
  const [comment, setComment] = useState("Просим проверить профиль компании.");

  async function onSubmit(event: React.FormEvent<HTMLFormElement>) {
    event.preventDefault();
    setPending(true);
    setNotice(null);

    try {
      await clientFetchJson(`/employer/companies/${companyId}/verification-requests`, {
        method: "POST",
        body: JSON.stringify({
          method: "official_website",
          submittedComment: comment,
          evidence: [{ evidenceType: "website_link", value }],
        }),
      });
      setNotice({ tone: "success", text: "Запрос на верификацию отправлен." });
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
        <span>Ссылка на подтверждающий ресурс</span>
        <input value={value} onChange={(event) => setValue(event.target.value)} required />
      </label>
      <label>
        <span>Комментарий</span>
        <textarea rows={3} value={comment} onChange={(event) => setComment(event.target.value)} />
      </label>
      <button type="submit" className="button button--primary" disabled={pending}>
        {pending ? "Отправляем..." : "Отправить на проверку"}
      </button>
      <FormNotice notice={notice} />
    </form>
  );
}

export function CuratorCreateForm({
  isAdmin,
}: {
  isAdmin: boolean;
}) {
  const router = useRouter();
  const [pending, setPending] = useState(false);
  const [notice, setNotice] = useState<Notice>(null);
  const [form, setForm] = useState({
    email: "",
    password: "password123",
    displayName: "",
    fullName: "",
    reason: "Расширяем команду модерации.",
  });

  if (!isAdmin) {
    return (
      <div className="guard-card guard-card--soft">
        <h3>Доступ только для admin-curator</h3>
        <p>Текущий куратор может просматривать команду, но не управлять ею.</p>
      </div>
    );
  }

  async function onSubmit(event: React.FormEvent<HTMLFormElement>) {
    event.preventDefault();
    setPending(true);
    setNotice(null);

    try {
      await clientFetchJson<CuratorAccount>("/curator/admin/curators", {
        method: "POST",
        body: JSON.stringify(form),
      });
      setNotice({ tone: "success", text: "Новый куратор создан." });
      setForm({
        email: "",
        password: "password123",
        displayName: "",
        fullName: "",
        reason: "Расширяем команду модерации.",
      });
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
        <span>Email</span>
        <input value={form.email} onChange={(event) => setForm({ ...form, email: event.target.value })} />
      </label>
      <label>
        <span>Display name</span>
        <input
          value={form.displayName}
          onChange={(event) => setForm({ ...form, displayName: event.target.value })}
        />
      </label>
      <label>
        <span>Полное имя</span>
        <input value={form.fullName} onChange={(event) => setForm({ ...form, fullName: event.target.value })} />
      </label>
      <label>
        <span>Причина</span>
        <input value={form.reason} onChange={(event) => setForm({ ...form, reason: event.target.value })} />
      </label>
      <button type="submit" className="button button--primary" disabled={pending}>
        {pending ? "Создаём..." : "Создать куратора"}
      </button>
      <FormNotice notice={notice} />
    </form>
  );
}
