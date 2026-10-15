// API Contracts and Types for saas-base

// Common API Error
export interface AppErrorResponse {
  code: string;
  message: string;
}

// -------------------------------------------------------------
// Group A: /v1/notes & /v1/tasks
// Handled either as Protobuf-serialized or Handler DTO
// -------------------------------------------------------------

export const NoteStatus = {
  UNSPECIFIED: 0,
  DRAFT: 1,
  PUBLISHED: 2,
} as const;

export type NoteStatusCode = typeof NoteStatus[keyof typeof NoteStatus];

export interface ProtoTimestamp {
  seconds?: number;
  nanos?: number;
}

export interface RawNote {
  id: string;
  title: string;
  body?: string;
  status: NoteStatusCode | string;
  created_at?: ProtoTimestamp | string;
}

export interface NoteModel {
  id: string;
  title: string;
  body?: string;
  status: "DRAFT" | "PUBLISHED" | "UNSPECIFIED";
  createdAt?: Date;
}

export type Note = NoteModel;

export interface ListNotesResponseRaw {
  items?: RawNote[];
  total?: number;
}

export interface ListNotesResponse {
  items: NoteModel[];
  notes: NoteModel[];
  total: number;
  totalCount: number;
}

export interface CountResponse {
  count: number;
}

export interface RawTask {
  id: string;
  title: string;
  note_id?: string;
  note_title?: string;
  status?: string;
  created_at?: ProtoTimestamp | string;
}

export interface TaskModel {
  id: string;
  title: string;
  noteId?: string;
  noteTitle?: string;
  status: string;
  createdAt: Date;
}

export type Task = TaskModel;

// Helper transformers for Group A
export const fromProtoTs = (ts?: ProtoTimestamp | string | null): Date => {
  if (!ts) return new Date();
  if (typeof ts === "string") {
    const d = new Date(ts);
    return isNaN(d.getTime()) ? new Date() : d;
  }
  if (typeof ts === "object" && typeof ts.seconds === "number") {
    return new Date(ts.seconds * 1000 + (ts.nanos ? Math.floor(ts.nanos / 1e6) : 0));
  }
  return new Date();
};

export const normalizeNoteStatus = (status: NoteStatusCode | string): "DRAFT" | "PUBLISHED" | "UNSPECIFIED" => {
  if (typeof status === "number") {
    if (status === NoteStatus.PUBLISHED) return "PUBLISHED";
    if (status === NoteStatus.DRAFT) return "DRAFT";
    return "UNSPECIFIED";
  }
  if (typeof status === "string") {
    const upper = status.toUpperCase();
    if (upper === "PUBLISHED") return "PUBLISHED";
    if (upper === "DRAFT") return "DRAFT";
  }
  return "UNSPECIFIED";
};

export const transformNote = (raw: RawNote): NoteModel => ({
  id: raw.id,
  title: raw.title,
  body: raw.body,
  status: normalizeNoteStatus(raw.status),
  createdAt: fromProtoTs(raw.created_at),
});

export const transformTask = (raw: RawTask): TaskModel => ({
  id: raw.id,
  title: raw.title,
  noteId: raw.note_id,
  noteTitle: raw.note_title,
  status: raw.status || "ACTIVE",
  createdAt: fromProtoTs(raw.created_at),
});

// -------------------------------------------------------------
// Group B: /v1/tenants (Standard Go Structs)
// -------------------------------------------------------------

export type TenantStatus = "ACTIVE" | "SUSPENDED" | "DEACTIVATED";

export interface RawTenant {
  id: string;
  name: string;
  slug: string;
  status: TenantStatus;
  created_at: string;
  updated_at?: string;
}

export interface TenantModel {
  id: string;
  name: string;
  slug: string;
  status: TenantStatus;
  createdAt: Date;
}

export type Tenant = TenantModel;

export interface ListTenantsResponseRaw {
  items: RawTenant[];
  total: number;
}

export interface CreateTenantRequest {
  name: string;
  slug: string;
}

export interface UpdateTenantRequest {
  id: string;
  name?: string;
  status?: TenantStatus;
}

export const transformTenant = (raw: RawTenant): TenantModel => ({
  id: raw.id,
  name: raw.name,
  slug: raw.slug,
  status: raw.status,
  createdAt: new Date(raw.created_at),
});

// -------------------------------------------------------------
// Auth
// -------------------------------------------------------------

export interface LoginRequest {
  username_or_email: string;
  password: string;
  tenant_id?: string;
}

export interface AuthTokens {
  access_token: string;
  refresh_token: string;
  expires_in?: number;
}

// ---------------------------------------------------------------- users & roles

export type Gender =
  | "GENDER_UNSPECIFIED"
  | "GENDER_MALE"
  | "GENDER_FEMALE"
  | "GENDER_OTHER";

export interface RoleRef {
  id: string;
  name: string;
}

export interface RawUser {
  id: string;
  username: string;
  name?: string;
  email: string;
  email_confirmed: boolean;
  gender?: Gender;
  roles?: RoleRef[];
  created_at: string;
}

export interface UserModel {
  id: string;
  username: string;
  name: string;
  email: string;
  emailConfirmed: boolean;
  gender: Gender;
  roles: RoleRef[];
  createdAt: Date;
}

export const transformUser = (raw: RawUser): UserModel => ({
  id: raw.id,
  username: raw.username,
  name: raw.name ?? "",
  email: raw.email,
  emailConfirmed: raw.email_confirmed,
  gender: raw.gender ?? "GENDER_UNSPECIFIED",
  roles: raw.roles ?? [],
  createdAt: new Date(raw.created_at),
});

export interface ListUsersResponseRaw {
  items: RawUser[];
  total: number;
}

export interface CreateUserRequest {
  username: string;
  email: string;
  password: string;
  name?: string;
}

export interface Role {
  id: string;
  name: string;
  preserved: boolean;
}

export interface ListRolesResponseRaw {
  items: Role[];
}

export interface Permission {
  namespace: string;
  action: string;
  host_only?: boolean;
}

export interface PermissionListRaw {
  items: Permission[];
}

export const permissionKey = (p: Pick<Permission, "namespace" | "action">): string =>
  `${p.namespace}:${p.action}`;
