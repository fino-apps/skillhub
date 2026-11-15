import { useState, type FormEvent } from "react";
import { compareSkills, createVersion, previewImport } from "@/generated/api-client";
import { CompareResults, type Change } from "./compare-panel";
import type { SkillVersion } from "./skill-version";

type Skill = { id: string; repositoryUrl?: string; repositoryPath?: string; localPath?: string };
type Preview = { commit?: string; digest?: string; changes?: Change[] };

export default function VersionPanel({ skill, latest, onSaved }: { skill: Skill; latest?: SkillVersion; onSaved: () => void }) {
  const [ref, setRef] = useState(latest?.sourceRef ?? "main");
  const [notes, setNotes] = useState("");
  const [preview, setPreview] = useState<Preview | null>(null);
  const [pending, setPending] = useState<"preview" | "save" | null>(null);
  const [error, setError] = useState("");

  async function inspect(event: FormEvent<HTMLFormElement>) {
    event.preventDefault();
    setError("");
    setPreview(null);
    setPending("preview");
    try {
      if (skill.localPath && latest) {
        const result = await compareSkills<{ files?: Change[]; rightDigest: string }>({
          left: { versionId: latest.id }, right: { localPath: skill.localPath },
        });
        setPreview({ digest: result.rightDigest, changes: result.files ?? [] });
        return;
      }
      const source = { repositoryUrl: skill.repositoryUrl, skillPath: skill.repositoryPath ?? "", ref };
      const { resolvedCommitSha: commit } = await previewImport<{ resolvedCommitSha: string }>({ source });
      if (!latest || latest.commitSha === commit) {
        setPreview({ commit });
        return;
      }
      const result = await compareSkills<{ files?: Change[] }>({
        left: { versionId: latest.id }, right: { github: { ...source, ref: commit } },
      });
      setPreview({ commit, changes: result.files ?? [] });
    } catch {
      setError(skill.localPath ? "无法预览差异。请检查原目录和变更文件大小。" : "无法读取或比对版本。请检查 Git 引用和仓库访问权限。");
    } finally {
      setPending(null);
    }
  }

  async function save() {
    if (!skill.localPath && !preview?.commit) return;
    setPending("save");
    setError("");
    try {
      const version = skill.localPath
        ? { expectedDigest: preview?.digest, notes }
        : { sourceRef: ref, commitSha: preview?.commit, notes };
      await createVersion({ skillId: skill.id, version });
      onSaved();
    } catch (reason) {
      const conflict = reason instanceof Error && reason.message === "resource conflict";
      setError(conflict
        ? skill.localPath ? "已有相同内容的版本，或目录在预览后发生了变化。请重新预览。" : "该 commit 已收录。"
        : "保存失败，请检查来源后重试。");
    } finally {
      setPending(null);
    }
  }

  const sameVersion = preview && (skill.localPath ? preview.digest === latest?.contentDigest : preview.commit === latest?.commitSha);
  return <div className="github-intake">
    <form className="github-form" onSubmit={inspect}>
      {skill.localPath ? <p className="skillhub-hint library-notes">原目录 <code>{skill.localPath}</code></p>
        : <label>分支、标签或 commit<input value={ref} onChange={(event) => { setRef(event.target.value); setPreview(null); setError(""); }} maxLength={200} required disabled={!!pending} /></label>}
      <button className="secondary" type="submit" disabled={!!pending || (!!skill.localPath && !latest)}>{pending === "preview" ? "读取中…" : skill.localPath ? "与最近版本比对" : "预览版本"}</button>
    </form>
    {error && <p className="intake-error" role="alert">{error}</p>}
    {(skill.localPath || preview) && <div className="github-preview">
      {preview?.commit && <p>固定 commit <code>{preview.commit}</code></p>}
      {sameVersion ? <p role="status">与最近收录的版本相同。</p> : <>
        {preview?.changes && <CompareResults files={preview.changes} />}
        <label className="version-notes">版本备注<input value={notes} maxLength={4096} disabled={!!pending} onChange={(event) => setNotes(event.target.value)} /></label>
        <button type="button" disabled={!!pending} onClick={() => void save()}>{pending === "save" ? "保存中…" : "保存版本"}</button>
      </>}
    </div>}
  </div>;
}
