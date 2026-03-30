export const API_BASE =
  process.env.NEXT_PUBLIC_TRAMPLIN_API_URL ?? "http://127.0.0.1:8090/v1";

export type UserRole = "applicant" | "employer" | "curator";

export interface Meta {
  page: number;
  pageSize: number;
  totalItems: number;
  totalPages: number;
}

export interface ListResponse<T> {
  items: T[];
  meta?: Meta;
}

export interface LocationPoint {
  id: string;
  placeName: string | null;
  addressLine: string | null;
  city: string | null;
  region: string | null;
  country: string | null;
  postalCode: string | null;
  precision: "exact_address" | "city_only" | string;
  latitude: number | null;
  longitude: number | null;
}

export interface Tag {
  id: string;
  name: string;
  tagType: string;
  isActive: boolean;
  isSystem: boolean;
}

export interface Company {
  id: string;
  slug: string;
  legalName: string;
  brandName: string;
  description: string | null;
  industry: string | null;
  websiteUrl: string | null;
  verificationStatus: string;
  logoMediaId: string | null;
  bannerMediaId?: string | null;
  headquartersLocation: LocationPoint | null;
  socialLinks?: CompanyLink[];
  media?: MediaAsset[];
  createdAt: string;
  updatedAt: string;
}

export interface CompanyLink {
  id: string;
  platform: string;
  url: string;
}

export interface MediaAsset {
  id: string;
  title?: string | null;
  url?: string | null;
  mediaFileId?: string | null;
}

export interface VacancyDetails {
  employmentType?: string | null;
  experienceLevel?: string | null;
  salaryFrom?: number | null;
  salaryTo?: number | null;
  currency?: string | null;
}

export interface EventDetails {
  startAt?: string | null;
  endAt?: string | null;
  registrationDeadline?: string | null;
  capacity?: number | null;
}

export interface MentorProgramDetails {
  startAt?: string | null;
  endAt?: string | null;
  seatsCount?: number | null;
}

export interface OpportunityLink {
  id: string;
  linkType: string;
  title: string;
  url: string;
}

export interface Opportunity {
  id: string;
  slug: string;
  title: string;
  summary: string;
  description?: string | null;
  type: string;
  status: string;
  moderationStatus?: string;
  participationFormat: string;
  publishedAt: string | null;
  expiresAt: string | null;
  createdAt: string;
  updatedAt: string;
  contactEmail?: string | null;
  contactPhone?: string | null;
  company: Company;
  location: LocationPoint | null;
  tags: Tag[];
  vacancyDetails: VacancyDetails | null;
  eventDetails: EventDetails | null;
  mentorProgramDetails: MentorProgramDetails | null;
  links?: OpportunityLink[];
  media?: MediaAsset[];
  stats?: {
    viewsCount: number;
    updatedAt: string;
  };
}

export interface CurrentUser {
  id: string;
  email: string;
  displayName: string;
  role: UserRole;
  isActive: boolean;
  avatarMediaId: string | null;
  emailVerifiedAt: string | null;
  lastLoginAt: string | null;
  createdAt: string;
  updatedAt: string;
  curatorProfile: {
    isAdmin: boolean;
  } | null;
}

export interface ApplicantProfile {
  userId: string;
  firstName: string | null;
  lastName: string | null;
  middleName: string | null;
  universityName: string | null;
  programName: string | null;
  faculty: string | null;
  studyYear: number | null;
  graduationYear: number | null;
  city: string | null;
  about: string | null;
  resumeMediaId: string | null;
  updatedAt: string;
}

export interface ApplicantPrivacy {
  profileVisibility: string;
  resumeVisibility: string;
  applicationsVisibility: string;
  contactsVisibility: string;
  showCareerInterests: boolean;
}

export interface SocialLink {
  id: string;
  platform: string;
  url: string;
  isPublic: boolean;
}

export interface ApplicantCompact {
  userId: string;
  displayName: string;
  firstName: string | null;
  lastName: string | null;
  middleName: string | null;
  city: string | null;
  universityName: string | null;
  graduationYear: number | null;
  avatarMediaId: string | null;
  tags: Tag[];
}

export interface Application {
  id: string;
  status: string;
  appliedAt: string;
  updatedAt: string;
  coverLetter: string | null;
  applicantUserId: string;
  opportunity: Opportunity;
  applicant?: ApplicantCompact;
  history?: Array<{
    id: string;
    status: string;
    comment: string | null;
    createdAt: string;
  }>;
}

export interface SavedOpportunity {
  opportunityId: string;
  applicantUserId: string;
  createdAt: string;
  opportunity: Opportunity;
}

export interface SavedCompany {
  companyId: string;
  applicantUserId: string;
  createdAt: string;
  company: Company;
}

export interface Connection {
  id: string;
  status: string;
  createdAt: string;
  respondedAt: string | null;
  initiatorUserId: string;
  initiatorNote: string | null;
  otherApplicant: ApplicantCompact;
}

export interface EmployerProfile {
  userId: string;
  fullName: string | null;
  jobTitle: string | null;
  phone: string | null;
  createdAt: string;
  updatedAt: string;
}

export interface CompanyMembership {
  id: string;
  companyId: string;
  employerUserId: string;
  membershipStatus: string;
  memberRole: string;
  isPrimaryContact: boolean;
  approvedAt: string | null;
  createdAt: string;
  company: Company;
  employer: {
    userId: string;
    displayName: string;
    email: string;
    fullName: string | null;
    jobTitle: string | null;
    avatarMediaId: string | null;
    isActive: boolean;
  };
}

export interface VerificationRequest {
  id: string;
  companyId: string;
  method: string;
  status: string;
  submittedByUserId: string;
  submittedComment: string | null;
  reviewComment: string | null;
  reviewedByCuratorUserId: string | null;
  reviewedAt: string | null;
  companyVerificationStatus: string;
  createdAt: string;
  evidence: Array<{
    id: string;
    evidenceType: string;
    value: string | null;
    evidenceFileId: string | null;
    createdAt: string;
  }>;
}

export interface ModerationCase {
  id: string;
  targetType: string;
  targetId: string;
  status: string;
  reason: string;
  createdAt: string;
  submittedByUserId: string;
  assignedCuratorUserId: string | null;
  resolvedByCuratorUserId: string | null;
  resolvedAt: string | null;
  targetPreview: {
    id: string;
    slug: string | null;
    title: string;
    subtitle: string | null;
    type: string;
  };
}

export interface CuratorAccount {
  id: string;
  email: string;
  displayName: string;
  fullName: string;
  role: string;
  isActive: boolean;
  isAdmin: boolean;
  curatorProfile: {
    isAdmin: boolean;
  };
  createdAt: string;
  updatedAt: string;
}

export interface ApiErrorShape {
  code: string;
  message: string;
  details?: unknown;
}

export class ApiError extends Error {
  status: number;
  code: string;

  constructor(status: number, payload: ApiErrorShape) {
    super(payload.message);
    this.name = "ApiError";
    this.status = status;
    this.code = payload.code;
  }
}

function apiUrl(path: string) {
  return `${API_BASE}${path.startsWith("/") ? path : `/${path}`}`;
}

async function parseJson<T>(response: Response): Promise<T> {
  if (response.status === 204) {
    return null as T;
  }
  return (await response.json()) as T;
}

export async function clientFetchJson<T>(
  path: string,
  init?: RequestInit,
): Promise<T> {
  const requestHeaders = new Headers(init?.headers);

  if (init?.body && !requestHeaders.has("content-type")) {
    requestHeaders.set("content-type", "application/json");
  }

  const response = await fetch(apiUrl(path), {
    ...init,
    credentials: "include",
    headers: requestHeaders,
  });

  if (!response.ok) {
    const payload = await parseJson<ApiErrorShape>(response);
    throw new ApiError(response.status, payload);
  }

  return parseJson<T>(response);
}

export function formatDate(date: string | null | undefined) {
  if (!date) {
    return "Без даты";
  }

  return new Intl.DateTimeFormat("ru-RU", {
    day: "numeric",
    month: "long",
  }).format(new Date(date));
}

export function formatDateTime(date: string | null | undefined) {
  if (!date) {
    return "Не указано";
  }

  return new Intl.DateTimeFormat("ru-RU", {
    day: "numeric",
    month: "long",
    hour: "2-digit",
    minute: "2-digit",
  }).format(new Date(date));
}

export function formatMoney(details: VacancyDetails | null | undefined) {
  if (!details || (!details.salaryFrom && !details.salaryTo)) {
    return "Уровень дохода обсуждается";
  }

  const currency = details.currency ?? "RUB";
  const formatter = new Intl.NumberFormat("ru-RU");

  if (details.salaryFrom && details.salaryTo) {
    return `${formatter.format(details.salaryFrom)}-${formatter.format(details.salaryTo)} ${currency}`;
  }

  if (details.salaryFrom) {
    return `от ${formatter.format(details.salaryFrom)} ${currency}`;
  }

  return `до ${formatter.format(details.salaryTo ?? 0)} ${currency}`;
}

export function formatOpportunityType(type: string) {
  const mapping: Record<string, string> = {
    internship: "Стажировка",
    vacancy: "Вакансия",
    mentor_program: "Менторская программа",
    event: "Карьерное событие",
  };

  return mapping[type] ?? type;
}

export function formatParticipation(format: string) {
  const mapping: Record<string, string> = {
    remote: "Удалённо",
    hybrid: "Гибрид",
    offline: "Очно",
  };

  return mapping[format] ?? format;
}

export function formatApplicationStatus(status: string) {
  const mapping: Record<string, string> = {
    submitted: "Отправлена",
    reviewing: "На рассмотрении",
    reserve: "В резерве",
    accepted: "Принята",
    rejected: "Отклонена",
    withdrawn: "Отозвана",
  };

  return mapping[status] ?? status;
}

export function formatVerificationStatus(status: string) {
  const mapping: Record<string, string> = {
    verified: "Проверена",
    pending: "На проверке",
    rejected: "Нужны исправления",
    approved: "Подтверждена",
  };

  return mapping[status] ?? status;
}

export function composeLocation(location: LocationPoint | null | undefined) {
  if (!location) {
    return "Локация уточняется";
  }

  const parts = [location.city, location.region, location.addressLine].filter(Boolean);
  return parts.length > 0 ? parts.join(", ") : "Локация уточняется";
}

export function initials(name: string) {
  return name
    .split(/\s+/)
    .filter(Boolean)
    .slice(0, 2)
    .map((part) => part[0]?.toUpperCase() ?? "")
    .join("");
}

export function roleHomePath(role: UserRole | null | undefined) {
  if (role === "applicant") {
    return "/applicant";
  }
  if (role === "employer") {
    return "/employer";
  }
  if (role === "curator") {
    return "/curator";
  }
  return "/auth/register";
}
