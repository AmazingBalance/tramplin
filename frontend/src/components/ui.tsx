import Link from "next/link";
import type { ReactNode } from "react";

import {
  composeLocation,
  formatApplicationStatus,
  formatDate,
  formatMoney,
  formatOpportunityType,
  formatParticipation,
  formatVerificationStatus,
  type Application,
  type Company,
  type ModerationCase,
  type Opportunity,
  type Tag,
  type VerificationRequest,
} from "@/lib/api";

import { CompanyMark } from "@/components/company-mark";

export function SectionHeading({
  eyebrow,
  title,
  text,
  action,
}: {
  eyebrow?: string;
  title: string;
  text?: string;
  action?: ReactNode;
}) {
  return (
    <div className="section-heading">
      <div>
        {eyebrow ? <p className="eyebrow">{eyebrow}</p> : null}
        <h2>{title}</h2>
        {text ? <p>{text}</p> : null}
      </div>
      {action ? <div>{action}</div> : null}
    </div>
  );
}

export function MetricCard({
  label,
  value,
  tone = "default",
}: {
  label: string;
  value: ReactNode;
  tone?: "default" | "accent" | "soft";
}) {
  return (
    <div className={`metric-card metric-card--${tone}`}>
      <span>{label}</span>
      <strong>{value}</strong>
    </div>
  );
}

export function StatusBadge({ value }: { value: string }) {
  const tone =
    value === "active" ||
    value === "accepted" ||
    value === "approved" ||
    value === "verified"
      ? "good"
      : value === "pending" || value === "reviewing" || value === "planned"
        ? "warm"
        : value === "draft" || value === "reserve"
          ? "neutral"
          : "bad";

  const text =
    value === "reviewing"
      ? formatApplicationStatus(value)
      : value === "verified" || value === "approved" || value === "rejected"
        ? formatVerificationStatus(value)
        : value;

  return <span className={`status-badge status-badge--${tone}`}>{text}</span>;
}

export function TagStrip({ tags }: { tags: Tag[] }) {
  return (
    <div className="tag-strip">
      {tags.map((tag) => (
        <span key={tag.id} className="tag-chip">
          {tag.name}
        </span>
      ))}
    </div>
  );
}

export function OpportunityCard({
  opportunity,
  actions,
}: {
  opportunity: Opportunity;
  actions?: ReactNode;
}) {
  return (
    <article className="entity-card">
      <div className="entity-card__topline">
        <span className="entity-card__label">
          {formatOpportunityType(opportunity.type)}
        </span>
        <StatusBadge value={opportunity.status} />
      </div>
      <div className="entity-card__body">
        <div className="entity-card__brandline">
          <CompanyMark company={opportunity.company} size="inline" />
          <div>
            <Link href={`/opportunities/${opportunity.slug}`} className="entity-card__title">
              {opportunity.title}
            </Link>
            <p className="entity-card__meta">
              {opportunity.company.brandName} · {formatParticipation(opportunity.participationFormat)}
            </p>
          </div>
        </div>
        <p className="entity-card__text">{opportunity.summary}</p>
        <TagStrip tags={opportunity.tags} />
        <dl className="entity-card__facts">
          <div>
            <dt>Доход</dt>
            <dd>{formatMoney(opportunity.vacancyDetails)}</dd>
          </div>
          <div>
            <dt>Локация</dt>
            <dd>{composeLocation(opportunity.location)}</dd>
          </div>
          <div>
            <dt>Дата</dt>
            <dd>{formatDate(opportunity.publishedAt ?? opportunity.createdAt)}</dd>
          </div>
        </dl>
      </div>
      {actions ? <div className="entity-card__actions">{actions}</div> : null}
    </article>
  );
}

export function CompanyCard({
  company,
  actions,
}: {
  company: Company;
  actions?: ReactNode;
}) {
  return (
    <article className="entity-card entity-card--company">
      <div className="entity-card__topline">
        <CompanyMark company={company} size="compact" />
        <StatusBadge value={company.verificationStatus} />
      </div>
      <div className="entity-card__body">
        <div>
          <Link href={`/companies/${company.slug}`} className="entity-card__title">
            {company.brandName}
          </Link>
          <p className="entity-card__meta">{company.industry ?? "Сфера уточняется"}</p>
        </div>
        <p className="entity-card__text">
          {company.description ?? "Компания ещё дополняет профиль, но уже открыта к знакомствам."}
        </p>
        <dl className="entity-card__facts">
          <div>
            <dt>Город</dt>
            <dd>{composeLocation(company.headquartersLocation)}</dd>
          </div>
          <div>
            <dt>Сайт</dt>
            <dd>{company.websiteUrl ?? "Скоро появится"}</dd>
          </div>
        </dl>
      </div>
      {actions ? <div className="entity-card__actions">{actions}</div> : null}
    </article>
  );
}

export function EmptyState({
  title,
  text,
  action,
}: {
  title: string;
  text: string;
  action?: ReactNode;
}) {
  return (
    <div className="empty-state">
      <h3>{title}</h3>
      <p>{text}</p>
      {action ? <div>{action}</div> : null}
    </div>
  );
}

export function GuardCard({
  title,
  text,
  href,
  actionLabel,
}: {
  title: string;
  text: string;
  href: string;
  actionLabel: string;
}) {
  return (
    <div className="guard-card">
      <h2>{title}</h2>
      <p>{text}</p>
      <Link href={href} className="button button--primary">
        {actionLabel}
      </Link>
    </div>
  );
}

export function QueueCard({
  title,
  subtitle,
  href,
  meta,
}: {
  title: string;
  subtitle: string;
  href?: string;
  meta?: ReactNode;
}) {
  const content = (
    <div className="queue-card">
      <div>
        <h3>{title}</h3>
        <p>{subtitle}</p>
      </div>
      {meta ? <div>{meta}</div> : null}
    </div>
  );

  return href ? <Link href={href}>{content}</Link> : content;
}

export function ApplicationRow({ item }: { item: Application }) {
  return (
    <article className="list-row">
      <div>
        <p className="list-row__title">{item.opportunity.title}</p>
        <p className="list-row__meta">
          {item.opportunity.company.brandName} · {composeLocation(item.opportunity.location)}
        </p>
      </div>
      <div className="list-row__aside">
        <StatusBadge value={item.status} />
        <span>{formatDate(item.appliedAt)}</span>
      </div>
    </article>
  );
}

export function VerificationRow({ item }: { item: VerificationRequest }) {
  return (
    <article className="list-row">
      <div>
        <p className="list-row__title">{formatVerificationStatus(item.status)}</p>
        <p className="list-row__meta">
          Метод: {item.method} · Последнее изменение {formatDate(item.reviewedAt ?? item.createdAt)}
        </p>
      </div>
      <div className="list-row__aside">
        <StatusBadge value={item.status} />
      </div>
    </article>
  );
}

export function ModerationRow({ item }: { item: ModerationCase }) {
  return (
    <article className="list-row">
      <div>
        <p className="list-row__title">{item.targetPreview.title}</p>
        <p className="list-row__meta">{item.targetPreview.subtitle ?? item.reason}</p>
      </div>
      <div className="list-row__aside">
        <StatusBadge value={item.status} />
      </div>
    </article>
  );
}
