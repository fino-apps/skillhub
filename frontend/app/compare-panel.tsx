import { useEffect, useState, type FormEvent } from "react";
import ReactDiffViewer from "react-diff-viewer-continued";
import { compareSkills, listVersions } from "@/generated/api-client";

type Skill = { id: string; name: string; kind?: string };
type Version = { id: string; version: string };
type Source = { skillId: string; versionId: string; path: string };
type Change = { path: string; kind: string; binary?: boolean; leftText?: string; rightText?: string };

function SourcePicker({ label, skills, value, onChange, disabled }: { label: string; skills: Skill[]; value: Source; onChange: (source: Source) => void; disabled: boolean }) {
  const [versions, setVersions] = useState<Version[]>([]);
  const [error, setError] = useState("");

  useEffect(() => {
    if (!value.skillId) { setVersions([]); return; }
    const controller = new AbortController();
    setVersions([]);
    setError("");
    async function load() {
      const items: Version[] = [];
      let pageToken = "";
      do {
        const result = await listVersions<{ versions?: Version[]; nextPageToken?: string }>({ skillId: value.skillId, pageSize: 100, ...(pageToken ? { pageToken } : {}) }, { signal: controller.signal });
        items.push(...(result.versions ?? []));
        pageToken = result.nextPageToken ?? "";
      } while (pageToken && !controller.signal.aborted);
      if (!controller.signal.aborted) setVersions(items);
    }
    void load().catch(() => { if (!controller.signal.aborted) setError("版本加载失败"); });
    return () => controller.abort();
  }, [value.skillId]);

  return <fieldset className="compare-source" disabled={disabled}><legend>{label}</legend>
    <label>来源<select value={value.skillId} onChange={(event) => onChange({ skillId: event.target.value, versionId: "", path: value.path })}>
      <option value="">本地目录</option>
      {skills.map((skill) => <option key={skill.id} value={skill.id}>{skill.name}</option>)}
    </select></label>
    {value.skillId ? <label>版本<select value={value.versionId} onChange={(event) => onChange({ ...value, versionId: event.target.value })}><option value="">当前内容</option>{versions.map((version) => <option key={version.id} value={version.id}>{version.version}</option>)}</select></label> : <label>目录路径<input value={value.path} onChange={(event) => onChange({ ...value, path: event.target.value })} placeholder="/path/to/skill" required /></label>}
    {error && <span role="alert" className="intake-error">{error}</span>}
  </fieldset>;
}

function location(source: Source) {
  if (source.versionId) return { versionId: source.versionId };
  if (source.skillId) return { currentSkillId: source.skillId };
  return { localPath: source.path.trim() };
}

export default function ComparePanel({ skills }: { skills: Skill[] }) {
  const [left, setLeft] = useState<Source>({ skillId: "", versionId: "", path: "" });
  const [right, setRight] = useState<Source>({ skillId: "", versionId: "", path: "" });
  const [changes, setChanges] = useState<Change[] | null>(null);
  const [selectedPath, setSelectedPath] = useState("");
  const [busy, setBusy] = useState(false);
  const [error, setError] = useState("");
  const [dark, setDark] = useState(document.documentElement.dataset.theme === "dark");

  useEffect(() => {
    const observer = new MutationObserver(() => setDark(document.documentElement.dataset.theme === "dark"));
    observer.observe(document.documentElement, { attributes: true, attributeFilter: ["data-theme"] });
    return () => observer.disconnect();
  }, []);

  async function submit(event: FormEvent<HTMLFormElement>) {
    event.preventDefault();
    setBusy(true); setError(""); setChanges(null);
    try {
      const result = await compareSkills<{ files?: Change[] }>({ left: location(left), right: location(right) });
      const files = result.files ?? [];
      setChanges(files); setSelectedPath(files[0]?.path ?? "");
    } catch (reason) {
      setError(reason instanceof Error && reason.message !== "bad request" ? reason.message : "无法比对，请检查目录及文件大小。");
    } finally { setBusy(false); }
  }

  const selected = changes?.find((file) => file.path === selectedPath);
  return <div className="compare-panel">
    <form onSubmit={submit}>
      <div className="compare-sources"><SourcePicker label="原始版本" skills={skills} value={left} disabled={busy} onChange={(value) => { setLeft(value); setChanges(null); }} /><SourcePicker label="目标版本" skills={skills} value={right} disabled={busy} onChange={(value) => { setRight(value); setChanges(null); }} /></div>
      <button type="submit" disabled={busy}>{busy ? "比对中…" : "开始比对"}</button>
    </form>
    {error && <p className="intake-error" role="alert">{error}</p>}
    {changes && (changes.length === 0 ? <p className="skillhub-empty">两个 Skill 内容一致。</p> : <div className="compare-results"><div className="compare-files">{changes.map((file) => <button type="button" key={file.path} className={selectedPath === file.path ? "is-selected" : ""} onClick={() => setSelectedPath(file.path)}><span>{file.kind.endsWith("ADDED") ? "+" : file.kind.endsWith("DELETED") ? "−" : "~"}</span>{file.path}</button>)}</div><div className="compare-content">{selected?.binary ? <p>二进制文件已变化。</p> : selected && <ReactDiffViewer oldValue={selected.leftText ?? ""} newValue={selected.rightText ?? ""} splitView={false} useDarkTheme={dark} />}</div></div>)}
  </div>;
}
