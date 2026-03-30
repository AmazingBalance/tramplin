"use client";

import Image from "next/image";
import { useEffect, useMemo, useState } from "react";
import { createPortal } from "react-dom";

type LandingFeatureStory = {
  title: string;
  caption: string;
  imageSrc: string;
  imageAlt: string;
  modalTitle: string;
  intro: string;
  points: string[];
  impact: string;
};

export function LandingFeatureStories({
  items,
}: {
  items: LandingFeatureStory[];
}) {
  const [activeIndex, setActiveIndex] = useState<number | null>(null);

  const activeItem = useMemo(
    () => (activeIndex === null ? null : items[activeIndex] ?? null),
    [activeIndex, items],
  );

  useEffect(() => {
    if (activeIndex === null) {
      return;
    }

    const previousOverflow = document.body.style.overflow;
    document.body.style.overflow = "hidden";

    function onKeyDown(event: KeyboardEvent) {
      if (event.key === "Escape") {
        setActiveIndex(null);
      }
    }

    window.addEventListener("keydown", onKeyDown);

    return () => {
      document.body.style.overflow = previousOverflow;
      window.removeEventListener("keydown", onKeyDown);
    };
  }, [activeIndex]);

  const modal =
    activeItem && typeof document !== "undefined"
      ? createPortal(
          <div
            className="feature-modal"
            role="presentation"
            onClick={() => setActiveIndex(null)}
          >
            <div
              role="dialog"
              aria-modal="true"
              aria-labelledby="landing-feature-title"
              className="feature-modal__dialog"
              onClick={(event) => event.stopPropagation()}
            >
              <button
                type="button"
                className="feature-modal__close"
                onClick={() => setActiveIndex(null)}
                aria-label="Закрыть"
              >
                ×
              </button>
              <div className="feature-modal__media">
                <Image
                  src={activeItem.imageSrc}
                  alt={activeItem.imageAlt}
                  fill
                  sizes="(max-width: 900px) 100vw, 420px"
                  className="feature-modal__image"
                />
              </div>
              <div className="feature-modal__content">
                <p className="eyebrow">Как это работает</p>
                <h3 id="landing-feature-title">{activeItem.modalTitle}</h3>
                <p>{activeItem.intro}</p>
                <div className="feature-modal__points">
                  {activeItem.points.map((point) => (
                    <div key={point} className="feature-modal__point">
                      {point}
                    </div>
                  ))}
                </div>
                <div className="feature-modal__impact">
                  <strong>Чем это помогает</strong>
                  <p>{activeItem.impact}</p>
                </div>
              </div>
            </div>
          </div>,
          document.body,
        )
      : null;

  return (
    <>
      <div className="feature-story-grid">
        {items.map((item, index) => (
          <button
            key={item.title}
            type="button"
            className="feature-story-card"
            onClick={() => setActiveIndex(index)}
          >
            <div className="feature-story-card__art">
              <Image
                src={item.imageSrc}
                alt={item.imageAlt}
                fill
                sizes="(max-width: 900px) 100vw, 360px"
                className="feature-story-card__image"
              />
            </div>
            <div className="feature-story-card__copy">
              <strong>{item.title}</strong>
              <span>{item.caption}</span>
            </div>
          </button>
        ))}
      </div>
      {modal}
    </>
  );
}
