"use client";

import dynamic from "next/dynamic";
import Link from "next/link";
import { useDeferredValue, useEffect, useState } from "react";

import {
  clientFetchJson,
  composeLocation,
  type Company,
  type ListResponse,
  type Opportunity,
  type Tag,
  type UserRole,
} from "@/lib/api";
import { quickQueries } from "@/lib/content";

import { SaveCompanyButton, SaveOpportunityButton } from "@/components/user-actions";
import { OpportunityCard } from "@/components/ui";

type ExplorerView = "list" | "map" | "split";

const OpportunityMap = dynamic(
  () => import("@/components/opportunity-map").then((module) => module.OpportunityMap),
  {
    ssr: false,
    loading: () => (
      <div className="map-board map-board--loading">
        <div className="map-board__legend">
          <span>Карта возможностей</span>
          <small>Загружаем карту...</small>
        </div>
      </div>
    ),
  },
);

function storageKey(kind: "companies" | "opportunities") {
  return `tramplin_guest_${kind}`;
}

function readStoredIds(kind: "companies" | "opportunities") {
  if (typeof window === "undefined") {
    return [] as string[];
  }

  try {
    const raw = window.localStorage.getItem(storageKey(kind));
    return raw ? (JSON.parse(raw) as string[]) : [];
  } catch {
    return [];
  }
}

export function HomeExplorer({
  opportunities,
  companies,
  tags,
  currentRole,
}: {
  opportunities: Opportunity[];
  companies: Company[];
  tags: Tag[];
  currentRole: UserRole | null;
}) {
  const [query, setQuery] = useState("");
  const [selectedTag, setSelectedTag] = useState<string>("all");
  const [selectedFormat, setSelectedFormat] = useState<string>("all");
  const [selectedType, setSelectedType] = useState<string>("all");
  const [selectedSalary, setSelectedSalary] = useState<string>("all");
  const [view, setView] = useState<ExplorerView>("split");
  const [savedCompanyIds, setSavedCompanyIds] = useState<string[]>([]);
  const deferredQuery = useDeferredValue(query);

  useEffect(() => {
    let cancelled = false;

    async function hydrateSaved() {
      if (currentRole === "applicant") {
        try {
          const savedCompanies = await clientFetchJson<ListResponse<{ companyId: string }>>(
            "/me/saved-companies?page=1&pageSize=100",
          );
          if (!cancelled) {
            setSavedCompanyIds(savedCompanies.items.map((item) => item.companyId));
          }
          return;
        } catch {
          // fall back to guest storage
        }
      }

      if (!cancelled) {
        setSavedCompanyIds(readStoredIds("companies"));
      }
    }

    hydrateSaved();

    return () => {
      cancelled = true;
    };
  }, [currentRole]);

  const filteredOpportunities = opportunities.filter((opportunity) => {
    const haystack = [
      opportunity.title,
      opportunity.summary,
      opportunity.company.brandName,
      composeLocation(opportunity.location),
      ...opportunity.tags.map((tag) => tag.name),
    ]
      .join(" ")
      .toLowerCase();

    const matchesQuery = haystack.includes(deferredQuery.trim().toLowerCase());
    const matchesTag =
      selectedTag === "all" || opportunity.tags.some((tag) => tag.id === selectedTag);
    const matchesFormat =
      selectedFormat === "all" || opportunity.participationFormat === selectedFormat;
    const matchesType = selectedType === "all" || opportunity.type === selectedType;
    const visibleSalary =
      opportunity.vacancyDetails?.salaryTo ?? opportunity.vacancyDetails?.salaryFrom ?? 0;
    const matchesSalary =
      selectedSalary === "all"
        ? true
        : selectedSalary === "paid"
          ? visibleSalary > 0
          : visibleSalary >= Number(selectedSalary);

    return matchesQuery && matchesTag && matchesFormat && matchesType && matchesSalary;
  });

  const pinnedCompanies = companies.filter((company) => savedCompanyIds.includes(company.id));

  return (
    <div className="explorer">
      <div className="explorer__toolbar">
        <div className="explorer__search">
          <label htmlFor="explorer-search">Поиск по возможностям</label>
          <input
            id="explorer-search"
            value={query}
            onChange={(event) => setQuery(event.target.value)}
            placeholder="Например: Go, стажировка в Москве, карьерное событие"
          />
        </div>
        <div className="explorer__controls">
          <select value={selectedTag} onChange={(event) => setSelectedTag(event.target.value)}>
            <option value="all">Все навыки и теги</option>
            {tags.map((tag) => (
              <option key={tag.id} value={tag.id}>
                {tag.name}
              </option>
            ))}
          </select>
          <select value={selectedFormat} onChange={(event) => setSelectedFormat(event.target.value)}>
            <option value="all">Любой формат</option>
            <option value="remote">Удалённо</option>
            <option value="hybrid">Гибрид</option>
            <option value="offline">Очно</option>
          </select>
          <select value={selectedType} onChange={(event) => setSelectedType(event.target.value)}>
            <option value="all">Любой тип</option>
            <option value="internship">Стажировка</option>
            <option value="vacancy">Вакансия</option>
            <option value="mentor_program">Менторская программа</option>
            <option value="event">Карьерное событие</option>
          </select>
          <select value={selectedSalary} onChange={(event) => setSelectedSalary(event.target.value)}>
            <option value="all">Любой доход</option>
            <option value="paid">Только оплачиваемые</option>
            <option value="80000">От 80 000 RUB</option>
            <option value="120000">От 120 000 RUB</option>
          </select>
        </div>
      </div>

      <div className="explorer__statusline">
        <strong>{filteredOpportunities.length} в выдаче</strong>
        <span>Начни с запроса, затем уточни фильтрами и открой подходящие карточки.</span>
      </div>

      <div className="explorer__quick">
        {quickQueries.map((item) => (
          <button key={item} type="button" className="quick-chip" onClick={() => setQuery(item)}>
            {item}
          </button>
        ))}
      </div>

      <div className="explorer__modes">
        <button
          type="button"
          className={view === "list" ? "mode-toggle is-active" : "mode-toggle"}
          onClick={() => setView("list")}
        >
          Лента
        </button>
        <button
          type="button"
          className={view === "map" ? "mode-toggle is-active" : "mode-toggle"}
          onClick={() => setView("map")}
        >
          Карта
        </button>
        <button
          type="button"
          className={view === "split" ? "mode-toggle is-active" : "mode-toggle"}
          onClick={() => setView("split")}
        >
          Вместе
        </button>
      </div>

      <div className={`explorer__content explorer__content--${view}`}>
        {view !== "map" ? (
          <div className="explorer__list">
            {filteredOpportunities.length > 0 ? (
              filteredOpportunities.map((opportunity) => (
                <OpportunityCard
                  key={opportunity.id}
                  opportunity={opportunity}
                  actions={
                    <div className="card-action-row">
                      <Link
                        href={`/opportunities/${opportunity.slug}`}
                        className="button button--ghost"
                      >
                        Открыть карточку
                      </Link>
                      <SaveOpportunityButton
                        opportunityId={opportunity.id}
                        currentRole={currentRole}
                      />
                    </div>
                  }
                />
              ))
            ) : (
              <div className="empty-state explorer__empty">
                <h3>Ничего не найдено</h3>
                <p>Попробуй другой запрос или ослабь один из фильтров.</p>
              </div>
            )}
          </div>
        ) : null}

        {view !== "list" ? (
          <div
            className={
              view === "split" ? "explorer__map explorer__map--split" : "explorer__map"
            }
          >
            <OpportunityMap
              key={view}
              opportunities={filteredOpportunities}
              savedCompanyIds={savedCompanyIds}
            />
            <div className="map-side-panel">
              <h3>Сохранённые компании</h3>
              {pinnedCompanies.length > 0 ? (
                pinnedCompanies.map((company) => (
                  <div key={company.id} className="mini-company-card">
                    <div>
                      <strong>{company.brandName}</strong>
                      <span>{composeLocation(company.headquartersLocation)}</span>
                    </div>
                    <SaveCompanyButton companyId={company.id} currentRole={currentRole} />
                  </div>
                ))
              ) : (
                <p>
                  Когда сохранишь компанию, она появится здесь и будет заметно выделена в поиске.
                </p>
              )}
            </div>
          </div>
        ) : null}
      </div>
    </div>
  );
}
