import { lazy, Suspense, useCallback, useEffect, useRef, useState, type FormEvent, type ReactNode } from "react";
import { getCopyCommand, listSkills, listVersions, scanProject } from "@/generated/api-client";
import GitHubIntake from "./github-intake";
import IntakePanel from "./intake-panel";
import "./skillhub.css";

const ComparePanel = lazy(() => import("./compare-panel"));

type Skill = { id: string; name: string; description: string; category?: string; kind?: string; repositoryUrl?: string; repositoryPath?: string };
type ProjectSkill = { path: string; name: string; description: string };
type Version = { id: string; version: string; commitSha?: string; notes?: string };
type LibraryResponse = { skills?: Skill[]; nextPageToken?: string };
type ScanResult = { path: string; skills: ProjectSkill[] };

function message(reason: unknown, fallback: string) {
  return reason instanceof Error && reason.message ? reason.message : fallback;
}

function ActionDialog({ title, onClose, children, compact = false }: { title: string; onClose: () => void; children: ReactNode; compact?: boolean }) {
  const ref = useRef<HTMLDialogElement>(null);
  useEffect(() => { if (ref.current && !ref.current.open) ref.current.showModal(); }, []);
  return <dialog className={`skillhub-dialog${compact ? " skillhub-dialog-compact" : ""}`} ref={ref} onClose={onClose} aria-labelledby="dialog-title">
    <div className="dialog-header"><h2 id="dialog-title">{title}</h2><button type="button" className="dialog-close" aria-label="关闭" onClick={onClose}>×</button></div>
    <div className="dialog-body">{children}</div>
  </dialog>;
}

function AddSkill({ onComplete }: { onComplete: () => void }) {
  const [tab, setTab] = useState<"local" | "github">("local");
  const [path, setPath] = useState("");
  const [result, setResult] = useState<ScanResult | null>(null);
  const [error, setError] = useState("");
  const [busy, setBusy] = useState(false);

  async function scan(event: FormEvent<HTMLFormElement>) {
    event.preventDefault();
    const projectPath = path.trim();
    if (!projectPath || busy) return;
    setBusy(true); setError(""); setResult(null);
    try {
      const response = await scanProject<{ skills?: ProjectSkill[] }>({ projectPath });
      setResult({ path: projectPath, skills: response.skills ?? [] });
    } catch (reason) {
      const detail = message(reason, "无法连接后端服务");
      setError(detail === "bad request" ? "路径无效或无法访问" : detail === "Failed to fetch" ? "无法连接后端服务" : detail);
    } finally { setBusy(false); }
  }

  return <div>
    <div className="source-tabs" role="tablist" aria-label="Skill 来源">
      <button type="button" role="tab" aria-selected={tab === "local"} onClick={() => setTab("local")}>本地</button>
      <button type="button" role="tab" aria-selected={tab === "github"} onClick={() => setTab("github")}>GitHub</button>
    </div>
    <div hidden={tab !== "local"} role="tabpanel">
      <form className="skillhub-form" onSubmit={scan}>
        <label htmlFor="project-path">项目或 Skill 目录</label>
        <div className="skillhub-form-row"><input id="project-path" value={path} onChange={(event) => setPath(event.target.value)} placeholder="/path/to/project/skills" required /><button type="submit" disabled={busy}>{busy ? "扫描中…" : "扫描"}</button></div>
      </form>
      {error && <p className="intake-error" role="alert">{error}</p>}
      {result && (result.skills.length === 0 ? <p className="skillhub-empty">未发现 Skill，请检查路径。</p> : <IntakePanel key={result.path} basePath={result.path} candidates={result.skills} onComplete={onComplete} />)}
    </div>
    <div hidden={tab !== "github"} role="tabpanel"><GitHubIntake onComplete={onComplete} /></div>
  </div>;
}

export default function HomePage() {
  const [skills, setSkills] = useState<Skill[]>([]);
  const [nextPageToken, setNextPageToken] = useState("");
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState("");
  const [keyword, setKeyword] = useState("");
  const [selected, setSelected] = useState("");
  const [versions, setVersions] = useState<Version[]>([]);
  const [versionsError, setVersionsError] = useState("");
  const [versionsLoading, setVersionsLoading] = useState(false);
  const [action, setAction] = useState<"add" | "compare" | null>(null);
  const [generating, setGenerating] = useState("");
  const [commandError, setCommandError] = useState("");
  const [command, setCommand] = useState("");
  const [copyStatus, setCopyStatus] = useState("");

  const loadLibrary = useCallback(async (token = "", signal?: AbortSignal) => {
    setLoading(true); setError("");
    try {
      const result = await listSkills<LibraryResponse>({ pageSize: 100, ...(token ? { pageToken: token } : {}) }, { signal });
      if (signal?.aborted) return;
      setSkills((previous) => token ? [...previous, ...(result.skills ?? [])] : result.skills ?? []);
      setNextPageToken(result.nextPageToken ?? "");
    } catch (reason) {
      if (!signal?.aborted) setError(message(reason, "无法连接后端服务"));
    } finally { if (!signal?.aborted) setLoading(false); }
  }, []);

  useEffect(() => {
    const controller = new AbortController();
    void loadLibrary("", controller.signal);
    return () => controller.abort();
  }, [loadLibrary]);

  useEffect(() => {
    if (!selected) return;
    const controller = new AbortController();
    setVersions([]); setVersionsError(""); setVersionsLoading(true);
    listVersions<{ versions?: Version[] }>({ skillId: selected }, { signal: controller.signal })
      .then((result) => setVersions(result.versions ?? []))
      .catch((reason: unknown) => { if (!controller.signal.aborted) setVersionsError(message(reason, "无法加载版本")); })
      .finally(() => { if (!controller.signal.aborted) setVersionsLoading(false); });
    return () => controller.abort();
  }, [selected]);

  async function prepareCommand(skillId: string, versionId: string) {
    if (generating) return;
    setGenerating(versionId);
    setCommandError("");
    try {
      const result = await getCopyCommand<{ command: string }>({ skillId, versionId });
      if (!result.command) throw new Error("未返回复制命令");
      setCopyStatus("");
      setCommand(result.command);
    } catch (reason) {
      setCommandError(message(reason, "请稍后重试"));
    } finally { setGenerating(""); }
  }

  async function copy() {
    try {
      await navigator.clipboard.writeText(command);
      setCopyStatus("已复制");
    } catch {
      setCopyStatus("复制失败，请手动选择命令");
    }
  }

  const query = keyword.trim().toLocaleLowerCase();
  const visible = skills.filter((skill) => `${skill.name} ${skill.description} ${skill.category ?? ""}`.toLocaleLowerCase().includes(query));

  return <main className="skillhub-shell">
    <header className="skillhub-heading"><div><h1>Skill 库</h1><span className="skillhub-count">{skills.length} 个已加载</span></div><div className="library-actions"><button type="button" className="secondary" onClick={() => setAction("compare")}>比对 Skill</button><button id="project-scan" type="button" onClick={() => setAction("add")}>添加 Skill</button></div></header>
    <label className="skillhub-search"><span aria-hidden="true">⌕</span><span className="sr-only">搜索 Skill</span><input value={keyword} onChange={(event) => setKeyword(event.target.value)} placeholder="搜索 Skill" /></label>
    {nextPageToken && <p className="skillhub-hint">搜索范围为已加载的 Skill。</p>}
    {error && <div className="skillhub-alert" role="alert"><span>加载失败：{error}</span><button type="button" className="secondary" onClick={() => void loadLibrary(skills.length ? nextPageToken : "")} disabled={loading}>重试</button></div>}
    {loading && skills.length === 0 ? <p className="skillhub-empty" role="status">正在加载…</p> : !error && skills.length === 0 ? <div className="skillhub-empty"><strong>还没有 Skill</strong><p><button type="button" className="skillhub-text-button" onClick={() => setAction("add")}>添加第一个 Skill</button></p></div> : visible.length === 0 && !error ? <div className="skillhub-empty"><strong>没有匹配的 Skill</strong><p><button type="button" className="skillhub-text-button" onClick={() => setKeyword("")}>清除搜索</button></p></div> : <ul className="skillhub-list">{visible.map((skill) => <li key={skill.id}><div className="skillhub-list-top"><button type="button" className="skillhub-library-item" aria-expanded={selected === skill.id} onClick={() => setSelected(selected === skill.id ? "" : skill.id)}>{skill.name}</button><span className="skillhub-tag">{skill.kind === "SKILL_KIND_PUBLIC" ? "GitHub" : "本地"}</span></div><p>{skill.description}</p>{selected === skill.id && <div className="library-versions">{skill.repositoryUrl && <p className="library-origin">来源：{skill.repositoryUrl}{skill.repositoryPath ? ` / ${skill.repositoryPath}` : ""}</p>}<div className="library-version-row"><span className="library-version-label">版本</span>{versionsLoading ? <p>加载中…</p> : versionsError ? <p role="alert">加载失败：{versionsError}</p> : versions.length ? <ul>{versions.map((version) => <li key={version.id}><span>{version.version}</span>{version.commitSha && <code title={version.commitSha}>{version.commitSha.slice(0, 12)}</code>}{version.notes && <span>{version.notes}</span>}<button type="button" className="skillhub-text-button" aria-label={`复制 ${skill.name} ${version.version} 的命令`} disabled={!!generating} onClick={() => void prepareCommand(skill.id, version.id)}>{generating === version.id ? "生成中…" : "复制命令"}</button></li>)}</ul> : <p>暂无版本</p>}</div></div>}</li>)}</ul>}
    {nextPageToken && !error && <button className="secondary skillhub-more" type="button" disabled={loading} onClick={() => void loadLibrary(nextPageToken)}>{loading ? "加载中…" : "加载更多"}</button>}
    {commandError && <p className="intake-error" role="alert">无法生成命令：{commandError}</p>}
    {command && <ActionDialog title="复制命令" compact onClose={() => setCommand("")}>
      <p className="skillhub-hint copy-hint">在目标目录执行；不会覆盖同名目录。</p>
      <div className="copy-command">
        <pre tabIndex={0}><code>{command}</code></pre>
        <button type="button" className="copy-command-button" aria-label="复制命令" title={copyStatus === "已复制" ? "已复制" : "复制命令"} onClick={() => void copy()}>
          {copyStatus === "已复制"
            ? <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round" aria-hidden="true"><path d="m5 12 4 4L19 6" /></svg>
            : <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round" aria-hidden="true"><rect x="8" y="8" width="12" height="12" rx="2" /><path d="M16 8V6a2 2 0 0 0-2-2H6a2 2 0 0 0-2 2v8a2 2 0 0 0 2 2h2" /></svg>}
        </button>
      </div>
      <span className="sr-only" role="status">{copyStatus}</span>
      {copyStatus && copyStatus !== "已复制" && <p className="intake-error">{copyStatus}</p>}
    </ActionDialog>}
    {action === "add" && <ActionDialog title="添加 Skill" onClose={() => setAction(null)}><AddSkill onComplete={() => void loadLibrary()} /></ActionDialog>}
    {action === "compare" && <ActionDialog title="比对 Skill" onClose={() => setAction(null)}><Suspense fallback={<p role="status">加载中…</p>}><ComparePanel skills={skills} /></Suspense></ActionDialog>}
  </main>;
}
