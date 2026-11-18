export type SkillVersion = {
  id: string;
  name?: string;
  sourceRef?: string;
  commitSha?: string;
  upstreamCommit?: string;
  upstreamRepository?: string;
  upstreamPath?: string;
  contentDigest?: string;
  createTime?: string;
  notes?: string;
  available?: boolean;
};

export function versionDigest(version: SkillVersion) {
  return version.commitSha || version.contentDigest || "";
}

export function shortVersion(version: SkillVersion, versions: SkillVersion[] = []) {
  const digest = versionDigest(version);
  let length = 8;
  while (length < digest.length && versions.some((other) =>
    other.id !== version.id && versionDigest(other) !== digest && versionDigest(other).startsWith(digest.slice(0, length)),
  )) {
    length += 1;
  }
  return digest.slice(0, length) || "—";
}

export function versionLabel(version: SkillVersion) {
  const date = version.createTime ? new Date(version.createTime) : null;
  return date && !Number.isNaN(date.getTime())
    ? date.toLocaleString("zh-CN", {
      year: "numeric", month: "numeric", day: "numeric",
      hour: "2-digit", minute: "2-digit", second: "2-digit", hourCycle: "h23",
    })
    : "未记录日期";
}
