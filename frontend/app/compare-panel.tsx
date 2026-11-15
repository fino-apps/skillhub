import { useEffect, useState, type FormEvent } from "react";
import ReactDiffViewer from "react-diff-viewer-continued";
import { compareSkills, listVersions, scanProject } from "@/generated/api-client";
import { versionLabel, type SkillVersion } from "./skill-version";

type Skill = { id: string; name: string };
type Source = { skillId: string; versionId: string; path: string };
export type Change = { path: string; kind: string; binary?: boolean; leftText?: string; rightText?: string };

function LocalSource({ path, onChange }: { path: string; onChange: (path: string) => void }) {
  const [found, setFound] = useState<{ path: string; name: string }[]>([]);
  const [base, setBase] = useState("");
  const [scanning, setScanning] = useState(false);
  const [feedback, setFeedback] = useState("");

  async function scan() {
    const projectPath = path.trim();
    if (!projectPath) return;
    setScanning(true);
    setFeedback("");
    setFound([]);
    try {
      const result = await scanProject<{ skills?: { path: string; name: string }[] }>({ projectPath });
      const skills = result.skills ?? [];
      setFound(skills);
      setBase(projectPath);
      if (!skills.length) setFeedback("这个目录下没有找到 Skill。");
    } catch {
      setFeedback("无法扫描，请检查本机目录路径。");
    } finally {
      setScanning(false);
    }
  }

  return <div className="compare-local">
    <label>本机项目或 Skill 目录<input value={path} onChange={(event) => { onChange(event.target.value); setFound([]); setFeedback(""); }} placeholder="/path/to/project" required disabled={scanning} /></label>
    <button type="button" className="secondary" disabled={scanning || !path.trim()} onClick={() => void scan()}>{scanning ? "扫描中…" : "扫描项目"}</button>
    {found.length > 0 && <label>选择 Skill<select defaultValue="" onChange={(event) => {
      const relative = event.target.value;
      if (relative) onChange(relative === "." ? base : `${base.replace(/\/$/, "")}/${relative}`);
    }}>
      <option value="" disabled>选择扫描结果</option>
      {found.map((skill) => <option key={skill.path} value={skill.path}>{skill.name}</option>)}
    </select></label>}
    {feedback && <p className="skillhub-hint" role="status">{feedback}</p>}
  </div>;
}

function SourcePicker({ label, skills, value, onChange, disabled }: { label: string; skills: Skill[]; value: Source; onChange: (source: Source) => void; disabled: boolean }) {
  const [versions, setVersions] = useState<SkillVersion[]>([]);
  const [error, setError] = useState("");

  useEffect(() => {
    if (!value.skillId) { setVersions([]); return; }
    const controller = new AbortController();
    setVersions([]);
    setError("");
    async function load() {
      const items: SkillVersion[] = [];
      let pageToken = "";
      do {
        const result = await listVersions<{ versions?: SkillVersion[]; nextPageToken?: string }>({ skillId: value.skillId, pageSize: 100, ...(pageToken ? { pageToken } : {}) }, { signal: controller.signal });
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
    {value.skillId ? <label>版本<select value={value.versionId} onChange={(event) => onChange({ ...value, versionId: event.target.value })}><option value="">最近收录版本</option>{versions.map((version) => <option key={version.id} value={version.id}>{versionLabel(version)} · {(version.commitSha ?? version.contentDigest)?.slice(0, 8)}</option>)}</select></label> : <LocalSource path={value.path} onChange={(path) => onChange({ ...value, path })} />}
    {error && <span role="alert" className="intake-error">{error}</span>}
  </fieldset>;
}

function location(source: Source) {
  if (source.versionId) return { versionId: source.versionId };
  if (source.skillId) return { currentSkillId: source.skillId };
  return { localPath: source.path.trim() };
}

export default function ComparePanel({ skills, skillId = "" }: { skills: Skill[]; skillId?: string }) {
  const [left, setLeft] = useState<Source>({ skillId, versionId: "", path: "" });
  const [right, setRight] = useState<Source>({ skillId, versionId: "", path: "" });
  const [changes, setChanges] = useState<Change[] | null>(null);
  const [busy, setBusy] = useState(false);
  const [error, setError] = useState("");

  async function submit(event: FormEvent<HTMLFormElement>) {
    event.preventDefault();
    setBusy(true); setError(""); setChanges(null);
    try {
      const result = await compareSkills<{ files?: Change[] }>({ left: location(left), right: location(right) });
      setChanges(result.files ?? []);
    } catch (reason) {
      setError(reason instanceof Error && reason.message !== "bad request" ? reason.message : "无法比对，请检查目录及文件大小。");
    } finally { setBusy(false); }
  }

  return <div className="compare-panel">
    <form onSubmit={submit}>
      <div className="compare-sources"><SourcePicker label="原始版本" skills={skills} value={left} disabled={busy} onChange={(value) => { setLeft(value); setChanges(null); }} /><SourcePicker label="目标版本" skills={skills} value={right} disabled={busy} onChange={(value) => { setRight(value); setChanges(null); }} /></div>
      <button type="submit" disabled={busy}>{busy ? "比对中…" : "开始比对"}</button>
    </form>
    {error && <p className="intake-error" role="alert">{error}</p>}
    {changes && <CompareResults files={changes} />}
  </div>;
}

export function CompareResults({ files }: { files: Change[] }) {
  const [selectedPath, setSelectedPath] = useState(files[0]?.path ?? "");
  const [dark, setDark] = useState(document.documentElement.dataset.theme === "dark");

  useEffect(() => {
    const observer = new MutationObserver(() => setDark(document.documentElement.dataset.theme === "dark"));
    observer.observe(document.documentElement, { attributes: true, attributeFilter: ["data-theme"] });
    return () => observer.disconnect();
  }, []);

  if (!files.length) return <p className="skillhub-empty">两个 Skill 内容一致。</p>;
  const selected = files.find((file) => file.path === selectedPath) ?? files[0];
  return <div className="compare-results">
    <div className="compare-files">{files.map((file) => <button type="button" key={file.path} className={selected.path === file.path ? "is-selected" : ""} onClick={() => setSelectedPath(file.path)}><span>{file.kind.endsWith("ADDED") ? "+" : file.kind.endsWith("DELETED") ? "−" : "~"}</span>{file.path}</button>)}</div>
    <div className="compare-content">{selected.binary ? <p>二进制文件已变化。</p> : <ReactDiffViewer oldValue={selected.leftText ?? ""} newValue={selected.rightText ?? ""} splitView={false} useDarkTheme={dark} />}</div>
  </div>;
}
