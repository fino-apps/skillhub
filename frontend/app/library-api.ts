import { listSkills, listVersions, selectLocalDirectory } from "@/generated/api-client";
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

export type Facets = { categories?: string[]; sources?: string[]; tags?: string[] };
export type LibraryPage = { skills?: Skill[]; nextPageToken?: string };
export type Candidate = { path?: string; name: string; description?: string };
export type ImportResult = { addedCount?: number; existingCount?: number; failures?: { path?: string; message: string }[] };

export const sourceNames: Record<string, string> = { folder: "本机目录", github: "GitHub", zip: "ZIP" };

export function sourceLabel(skill: Skill) {
  return skill.upstreamName || sourceNames[skill.importKind ?? ""] || "未知";
}

export function message(error: unknown) {
  return error instanceof Error ? error.message : "操作失败，请重试";
}

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
  return new URL(`/skillhub/v1/skills/${skillId}/versions/${versionId}/archive`, config.apiBase || window.location.origin).href;
}

export async function stageZip(file: File) {
  const form = new FormData();
  form.append("file", file);
  const response = await fetch(new URL("/skillhub/v1/uploads", config.apiBase || window.location.origin), { method: "POST", body: form });
  const result = await response.json() as { uploadId: string; skills?: Candidate[]; message?: string };
  if (!response.ok) throw new Error(result.message || "无法读取 ZIP");
  return result;
}

type ListSnapshot = { scope: string; skills: Skill[]; token: string; scroll: number; skillId: string; revision: number };
export const libraryNavigation: { returning: ListSnapshot | null; revision: number } = { returning: null, revision: 0 };
