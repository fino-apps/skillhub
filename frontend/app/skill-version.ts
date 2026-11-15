export type SkillVersion = {
  id: string;
  sourceRef?: string;
  commitSha?: string;
  contentDigest?: string;
  createTime?: string;
  notes?: string;
};

export function versionLabel(version: SkillVersion) {
  if (version.commitSha) return version.sourceRef ?? "GitHub";
  const created = version.createTime ? new Date(version.createTime) : null;
  return created && !Number.isNaN(created.getTime()) ? created.toLocaleString("zh-CN", { hour12: false }) : "本地快照";
}
