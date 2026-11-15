import { useState, type FormEvent } from "react";
import { importSkill, previewImport } from "@/generated/api-client";

type Source = { repositoryUrl: string; skillPath: string; ref: string };
type Candidate = { path?: string; name: string; description: string };
type Preview = { resolvedCommitSha: string; skills?: Candidate[] };

export default function GitHubIntake({ onComplete }: { onComplete: () => void }) {
  const [source, setSource] = useState<Source>({ repositoryUrl: "", skillPath: "", ref: "main" });
  const [collection, setCollection] = useState(false);
  const [preview, setPreview] = useState<Preview | null>(null);
  const [selected, setSelected] = useState<string[]>([]);
  const [pending, setPending] = useState<"preview" | "save" | null>(null);
  const [feedback, setFeedback] = useState("");
  const [error, setError] = useState("");

  function resetPreview() {
    setPreview(null);
    setSelected([]);
    setFeedback("");
    setError("");
  }

  function change(field: keyof Source, value: string) {
    setSource((current) => ({ ...current, [field]: value }));
    resetPreview();
  }

  function toggle(path: string) {
    setSelected((current) => current.includes(path) ? current.filter((item) => item !== path) : [...current, path]);
  }

  async function inspect(event: FormEvent<HTMLFormElement>) {
    event.preventDefault();
    resetPreview();
    setPending("preview");
    try {
      const result = await previewImport<Preview>({ source, scanCollection: collection });
      setPreview(result);
      setSelected((result.skills ?? []).slice(0, 100).map((skill) => skill.path ?? ""));
    } catch {
      setError("无法读取 Skill。请检查仓库、路径、引用和本机 Git 凭据。");
    } finally {
      setPending(null);
    }
  }

  async function save() {
    if (!preview || !selected.length) return;
    setPending("save");
    setFeedback("");
    setError("");
    try {
      const result = await importSkill<{ addedCount?: number; existingCount?: number }>({
        repositoryUrl: source.repositoryUrl, sourceRef: source.ref, commitSha: preview.resolvedCommitSha, skillPaths: selected,
      });
      resetPreview();
      setFeedback(`已收录 ${result.addedCount ?? 0} 个，已存在 ${result.existingCount ?? 0} 个。`);
      onComplete();
    } catch {
      setError("收录失败。请确认预览的 commit 仍可读取后重试。");
    } finally {
      setPending(null);
    }
  }

  return <div className="github-intake">
    <form onSubmit={inspect} className="github-form">
      <label>仓库地址<input type="url" disabled={!!pending} value={source.repositoryUrl} onChange={(event) => change("repositoryUrl", event.target.value)} placeholder="https://github.com/owner/repo" required /></label>
      <div>
        <label>仓库内路径<input disabled={!!pending || collection} value={source.skillPath} onChange={(event) => change("skillPath", event.target.value)} placeholder="skills/playwright（根目录可留空）" /></label>
        <label>分支、标签或 commit<input disabled={!!pending} value={source.ref} onChange={(event) => change("ref", event.target.value)} maxLength={200} required /></label>
      </div>
      <label className="github-checkbox"><input type="checkbox" checked={collection} disabled={!!pending} onChange={(event) => { setCollection(event.target.checked); resetPreview(); }} />批量扫描仓库的 skills/ 目录</label>
      <button className="secondary" type="submit" disabled={!!pending}>{pending === "preview" ? "读取中…" : "预览来源"}</button>
    </form>
    {error && <p className="intake-error" role="alert">{error}</p>}
    {feedback && <p className="skillhub-hint" role="status">{feedback}</p>}
    {preview && <div className="github-preview">
      <p>固定 commit <code>{preview.resolvedCommitSha}</code></p>
      {preview.skills?.length ? <>
        <ul className="github-candidates">{preview.skills.map((skill) => {
          const path = skill.path ?? "";
          const checked = selected.includes(path);
          return <li key={path}><label className="github-checkbox">
            <input type="checkbox" checked={checked} disabled={!!pending || (!checked && selected.length >= 100)} onChange={() => toggle(path)} />
            <span><strong>{skill.name}</strong> <small>{path || "仓库根目录"}</small><span className="candidate-description">{skill.description}</span></span>
          </label></li>;
        })}</ul>
        {preview.skills.length > 100 && <p className="skillhub-hint">每次最多收录 100 个。</p>}
        <button type="button" onClick={() => void save()} disabled={!!pending || !selected.length}>{pending === "save" ? "收录中…" : `收录所选（${selected.length}）`}</button>
      </> : <p role="status">skills/ 下没有可收录的 Skill。</p>}
    </div>}
  </div>;
}
