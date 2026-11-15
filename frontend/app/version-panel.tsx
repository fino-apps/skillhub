import { useState, type FormEvent } from "react";
import { compareSkills, createVersion, previewImport } from "@/generated/api-client";
import { CompareResults, type Change } from "./compare-panel";

type Skill = { id: string; repositoryUrl: string; repositoryPath?: string };
type Version = { id: string; sourceRef: string; commitSha: string };

export default function VersionPanel({ skill, latest, onSaved }: { skill: Skill; latest?: Version; onSaved: () => void }) {
  const [ref, setRef] = useState(latest?.sourceRef ?? "main");
  const [notes, setNotes] = useState("");
  const [preview, setPreview] = useState<{ commit: string; changes?: Change[] } | null>(null);
  const [pending, setPending] = useState<"preview" | "save" | null>(null);
  const [error, setError] = useState("");

  async function inspect(event: FormEvent<HTMLFormElement>) {
    event.preventDefault();
    setError("");
    setPreview(null);
    setPending("preview");
    const source = { repositoryUrl: skill.repositoryUrl, skillPath: skill.repositoryPath ?? "", ref };
    try {
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
      setError("无法读取或比对版本。请检查 Git 引用和仓库访问权限。");
    } finally {
      setPending(null);
    }
  }

  async function save() {
    if (!preview) return;
    setPending("save");
    setError("");
    try {
      await createVersion({ skillId: skill.id, version: { sourceRef: ref, commitSha: preview.commit, notes } });
      onSaved();
    } catch (reason) {
      setError(reason instanceof Error && reason.message === "resource conflict" ? "该 commit 已收录。" : "保存失败，请重新预览后重试。");
    } finally {
      setPending(null);
    }
  }

  return <div className="github-intake">
    <form className="github-form" onSubmit={inspect}>
      <label>分支、标签或 commit<input value={ref} onChange={(event) => { setRef(event.target.value); setPreview(null); setError(""); }} maxLength={200} required disabled={!!pending} /></label>
      <button className="secondary" type="submit" disabled={!!pending}>{pending === "preview" ? "读取中…" : "预览版本"}</button>
    </form>
    {error && <p className="intake-error" role="alert">{error}</p>}
    {preview && <div className="github-preview">
      <p>固定 commit <code>{preview.commit}</code></p>
      {latest?.commitSha === preview.commit ? <p role="status">与最近收录的版本相同。</p> : <>
        {preview.changes && <CompareResults files={preview.changes} />}
        <label className="version-notes">版本备注<input value={notes} maxLength={4096} disabled={!!pending} onChange={(event) => setNotes(event.target.value)} /></label>
        <button type="button" disabled={!!pending} onClick={() => void save()}>{pending === "save" ? "保存中…" : "保存版本"}</button>
      </>}
    </div>}
  </div>;
}
