"use client";

import Link from "next/link";
import { useEffect, useMemo, useRef, useState } from "react";
import { MapContainer, Marker, TileLayer, useMap } from "react-leaflet";
import L from "leaflet";

import {
  composeLocation,
  formatOpportunityType,
  formatParticipation,
  type Opportunity,
} from "@/lib/api";

type MarkerOpportunity = Opportunity & {
  markerLat: number;
  markerLng: number;
  markerPrecision: "exact" | "city";
};

const CITY_COORDINATES: Record<string, [number, number]> = {
  moscow: [55.7558, 37.6173],
  "moscow city": [55.7558, 37.6173],
  "москва": [55.7558, 37.6173],
  "saint petersburg": [59.9343, 30.3351],
  "st. petersburg": [59.9343, 30.3351],
  "санкт-петербург": [59.9343, 30.3351],
  "петербург": [59.9343, 30.3351],
  kazan: [55.7961, 49.1064],
  "казань": [55.7961, 49.1064],
  novosibirsk: [55.0084, 82.9357],
  "новосибирск": [55.0084, 82.9357],
  yekaterinburg: [56.8389, 60.6057],
  ekaterinburg: [56.8389, 60.6057],
  "екатеринбург": [56.8389, 60.6057],
  tomsk: [56.4846, 84.9477],
  "томск": [56.4846, 84.9477],
  innopolis: [55.7522, 48.7446],
  "иннополис": [55.7522, 48.7446],
  samara: [53.1959, 50.1008],
  "самара": [53.1959, 50.1008],
  krasnodar: [45.0355, 38.9753],
  "краснодар": [45.0355, 38.9753],
  omsk: [54.9885, 73.3242],
  "омск": [54.9885, 73.3242],
  vladivostok: [43.1155, 131.8855],
  "владивосток": [43.1155, 131.8855],
  "nizhny novgorod": [56.3269, 44.0059],
  "нижний новгород": [56.3269, 44.0059],
};

function normalizeCityKey(value: string | null | undefined) {
  return value?.trim().toLowerCase() ?? "";
}

function resolveMarkerPoint(opportunity: Opportunity) {
  const lat = opportunity.location?.latitude;
  const lng = opportunity.location?.longitude;
  const precision = opportunity.location?.precision;

  if (typeof lat === "number" && typeof lng === "number") {
    return {
      markerLat: lat,
      markerLng: lng,
      markerPrecision: precision === "exact_address" ? ("exact" as const) : ("city" as const),
    };
  }

  const cityKey = normalizeCityKey(opportunity.location?.city);
  const fallback = CITY_COORDINATES[cityKey];

  if (fallback) {
    return {
      markerLat: fallback[0],
      markerLng: fallback[1],
      markerPrecision: "city" as const,
    };
  }

  return null;
}

function createMarkerIcon({
  isPinned,
  isActive,
}: {
  isPinned: boolean;
  isActive: boolean;
}) {
  return L.divIcon({
    className: `map-pin${isPinned ? " is-pinned" : ""}${isActive ? " is-active" : ""}`,
    html: '<span class="map-pin__core"></span>',
    iconSize: [22, 22],
    iconAnchor: [11, 11],
  });
}

function KeepMapSized() {
  const map = useMap();

  useEffect(() => {
    const refresh = () => {
      map.invalidateSize({ pan: false, debounceMoveend: true });
    };

    const frame = window.requestAnimationFrame(refresh);
    const observer = new ResizeObserver(refresh);
    observer.observe(map.getContainer());

    return () => {
      window.cancelAnimationFrame(frame);
      observer.disconnect();
    };
  }, [map]);

  return null;
}

function FitMapToMarkers({ items }: { items: MarkerOpportunity[] }) {
  const map = useMap();

  useEffect(() => {
    map.invalidateSize({ pan: false, debounceMoveend: true });

    if (items.length === 0) {
      map.setView([56.5, 52], 4);
      return;
    }

    if (items.length === 1) {
      map.setView([items[0].markerLat, items[0].markerLng], 11);
      return;
    }

    const bounds = L.latLngBounds(items.map((item) => [item.markerLat, item.markerLng]));
    const size = map.getSize();
    const rightPadding = Math.min(360, Math.max(80, Math.round(size.x * 0.24)));
    const leftPadding = Math.min(72, Math.max(40, Math.round(size.x * 0.08)));
    const verticalPadding = Math.min(88, Math.max(48, Math.round(size.y * 0.12)));

    map.fitBounds(bounds, {
      paddingTopLeft: [leftPadding, verticalPadding],
      paddingBottomRight: [rightPadding, verticalPadding],
    });
  }, [items, map]);

  return null;
}

export function OpportunityMap({
  opportunities,
  savedCompanyIds,
}: {
  opportunities: Opportunity[];
  savedCompanyIds: string[];
}) {
  const closeTimerRef = useRef<number | null>(null);
  const [activeId, setActiveId] = useState<string | null>(null);

  const items = useMemo(
    () =>
      opportunities
        .map((opportunity) => {
          const point = resolveMarkerPoint(opportunity);
          return point ? { ...opportunity, ...point } : null;
        })
        .filter((item): item is MarkerOpportunity => item !== null),
    [opportunities],
  );

  const visibleActiveId = activeId && items.some((item) => item.id === activeId) ? activeId : null;
  const activeItem = items.find((item) => item.id === visibleActiveId) ?? null;

  useEffect(() => {
    return () => {
      if (closeTimerRef.current) {
        window.clearTimeout(closeTimerRef.current);
      }
    };
  }, []);

  function clearCloseTimer() {
    if (closeTimerRef.current) {
      window.clearTimeout(closeTimerRef.current);
      closeTimerRef.current = null;
    }
  }

  function activate(id: string) {
    clearCloseTimer();
    setActiveId(id);
  }

  function scheduleClose(id: string) {
    clearCloseTimer();
    closeTimerRef.current = window.setTimeout(() => {
      setActiveId((current) => (current === id ? null : current));
    }, 180);
  }

  if (items.length === 0) {
    return (
      <div className="map-board map-board--empty">
        <div className="map-board__legend">
          <span>Карта возможностей</span>
          <small>Для текущего набора фильтров нет записей с координатами.</small>
        </div>
      </div>
    );
  }

  return (
    <div className="map-board">
      <div className="map-board__legend">
        <span>Карта возможностей</span>
        <small>Точки ставятся по адресу, а если его нет, то по координатам города.</small>
      </div>

      <MapContainer center={[56.5, 52]} zoom={4} scrollWheelZoom className="map-surface">
        <KeepMapSized />
        <TileLayer
          attribution='&copy; OpenStreetMap contributors &copy; CARTO'
          detectRetina
          subdomains={["a", "b", "c", "d"]}
          url="https://{s}.basemaps.cartocdn.com/light_all/{z}/{x}/{y}{r}.png"
        />
        <FitMapToMarkers items={items} />
        {items.map((item) => {
          const isPinned = savedCompanyIds.includes(item.company.id);
          const isActive = item.id === visibleActiveId;

          return (
            <Marker
              key={item.id}
              position={[item.markerLat, item.markerLng]}
              icon={createMarkerIcon({ isPinned, isActive })}
              eventHandlers={{
                mouseover: () => activate(item.id),
                mouseout: () => scheduleClose(item.id),
                click: () => activate(item.id),
              }}
            />
          );
        })}
      </MapContainer>

      {activeItem ? (
        <div
          className="map-hover-card"
          onMouseEnter={() => activate(activeItem.id)}
          onMouseLeave={() => scheduleClose(activeItem.id)}
        >
          <div className="map-hover-card__top">
            <span className="entity-card__label">{formatOpportunityType(activeItem.type)}</span>
            <span className="map-hover-card__format">
              {formatParticipation(activeItem.participationFormat)}
            </span>
          </div>
          <div className="map-hover-card__body">
            <strong>{activeItem.title}</strong>
            <span>{activeItem.company.brandName}</span>
            <small>{composeLocation(activeItem.location)}</small>
            <small>
              {activeItem.markerPrecision === "exact"
                ? "Точка поставлена по координатам адреса."
                : "Точка поставлена по координатам города."}
            </small>
          </div>
          <Link href={`/opportunities/${activeItem.slug}`} className="button button--ghost">
            Открыть
          </Link>
        </div>
      ) : null}
    </div>
  );
}
