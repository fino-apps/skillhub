import { listSkills, listVersions, selectLocalDirectory, uploadZip } from "@/generated/api-client";
import { apiRoutes } from "@/generated/api-routes";
import { config } from "@/lib/config";
import type { SkillVersion } from "./skill-version";

export type Skill = {
  id: string;
  name: string;
  description?: string;
  purpose?: string;
  category?: string;
  tags?: string[];
  notes?: string;
  importKind?: string;
  upstreamName?: string;
  upstreamUrl?: string;
  repositoryUrl?: string;
  repositoryPath?: string;
  localPath?: string;
  defaultVersion?: SkillVersion;
};

export type Facets = { categories?: string[]; sources?: string[]; sourceNames?: string[]; tags?: string[] };
export type LibraryPage = { skills?: Skill[]; nextPageToken?: string };
export type Candidate = { path?: string; name: string; description?: string };
export type ImportResult = { addedCount?: number; existingCount?: number; failures?: { path?: string; message: string }[] };

export const sourceNames: Record<string, string> = { folder: "本机目录", directory_upload: "本机目录", github: "GitHub", zip: "ZIP" };

export function importSourceName(kind = "") {
  return Object.hasOwn(sourceNames, kind) ? sourceNames[kind] : kind || "未知";
}

export function sourceURL(skill: Skill) {
  const url = skill.upstreamUrl?.trim();
  return url && /^https?:\/\//i.test(url) && URL.canParse(url) ? url : "";
}

export function sourceLabel(skill: Skill) {
  return skill.upstreamName?.trim() || sourceURL(skill).replace(/^https?:\/\//i, "") || importSourceName(skill.importKind);
}

export function message(error: unknown) {
  return error instanceof Error ? error.message : "操作失败，请重试";
}

const desktopPicker = import.meta.env.VITE_DESKTOP_DIRECTORY_PICKER;
const loopbackHosts = ["localhost", "127.0.0.1", "[::1]"];

export const canPickDirectory = desktopPicker === undefined
  ? import.meta.env.DEV && loopbackHosts.includes(window.location.hostname)
    && loopbackHosts.includes(new URL(config.apiBase || "/", window.location.href).hostname)
  : desktopPicker === "true";

export const backendDirectoryLabel = canPickDirectory ? "本机目录" : "后端目录（容器内绝对路径）";

export async function pickDirectory() {
  const result = await selectLocalDirectory<{ path?: string }>();
  return result.path ?? "";
}

export function localCandidate(root: string, relative?: string) {
  return !relative || relative === "." ? root : root.replace(/[/\\]+$/, "") + "/" + relative;
}

export function tagsFrom(value: string) {
  return [...new Set(value.split(/[,，\n]/).map((tag) => tag.trim()).filter(Boolean))];
}

export async function fetchAllSkills(signal?: AbortSignal) {
  const skills: Skill[] = [];
  let pageToken = "";
  do {
    const result = await listSkills<LibraryPage>({ pageSize: 100, pageToken }, { signal });
    skills.push(...(result.skills ?? []));
    pageToken = result.nextPageToken ?? "";
  } while (pageToken);
  return skills;
}

export async function fetchAllVersions(skillId: string, signal: AbortSignal) {
  const versions: SkillVersion[] = [];
  let pageToken = "";
  do {
    const result = await listVersions<{ versions?: SkillVersion[]; nextPageToken?: string }>({ skillId, pageSize: 100, pageToken }, { signal });
    versions.push(...(result.versions ?? []));
    pageToken = result.nextPageToken ?? "";
  } while (pageToken);
  return versions;
}

export function archiveURL(skillId: string, versionId: string) {
  const path = apiRoutes.downloadZip.path
    .replace("{skill_id}", encodeURIComponent(skillId))
    .replace("{version_id}", encodeURIComponent(versionId));
  const url = new URL(path, config.apiBase || window.location.origin);
  url.searchParams.set("includeRootDirectory", "true");
  return url.href;
}

export function stageZip(file: File) {
  return uploadZip<{ uploadId: string; skills?: Candidate[] }>({ file });
}

type ListSnapshot = { scope: string; skills: Skill[]; token: string; scroll: number; skillId: string; revision: number };
export const libraryNavigation: { returning: ListSnapshot | null; revision: number } = { returning: null, revision: 0 };
