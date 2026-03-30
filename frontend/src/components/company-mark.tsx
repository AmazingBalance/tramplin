import { initials, type Company } from "@/lib/api";

type CompanyIdentity = Pick<Company, "slug" | "brandName">;

export function CompanyMark({
  company,
  size = "compact",
}: {
  company: CompanyIdentity;
  size?: "inline" | "compact" | "full";
}) {
  const slugClass = `company-mark--${company.slug}`;

  if (company.slug === "avito-tech") {
    return (
      <span
        className={`company-mark company-mark--dots ${slugClass} company-mark--${size}`}
        aria-label={company.brandName}
        title={company.brandName}
      >
        <span className="company-mark__dots">
          <span className="company-mark__dot company-mark__dot--green" />
          <span className="company-mark__dot company-mark__dot--blue" />
          <span className="company-mark__dot company-mark__dot--red" />
          <span className="company-mark__dot company-mark__dot--yellow" />
        </span>
        {size === "full" ? <strong>{company.brandName}</strong> : null}
      </span>
    );
  }

  const glyph =
    company.slug === "yandex"
      ? "Я"
      : company.slug === "ozon-tech"
        ? "OZ"
        : company.slug === "t-bank"
          ? "T"
          : company.slug === "vk"
            ? "VK"
            : company.slug === "mts-digital"
              ? "MTS"
              : company.slug === "sbertech"
                ? "S"
                : initials(company.brandName);

  return (
    <span
      className={`company-mark ${slugClass} company-mark--${size}`}
      aria-label={company.brandName}
      title={company.brandName}
    >
      <span className="company-mark__glyph">{glyph}</span>
      {size === "full" ? <strong>{company.brandName}</strong> : null}
    </span>
  );
}
