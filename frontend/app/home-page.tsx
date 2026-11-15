import { useCallback, useEffect, useState, type FormEvent } from "react";
import { listSkills, listVersions, scanProject } from "@/generated/api-client";
import IntakePanel from "./intake-panel";
import GitHubIntake from "./github-intake";
import "./skillhub.css";

type Skill = { id: string; name: string; description: string; category?: string; kind?: string; repositoryUrl?: string; repositoryPath?: string };
type ProjectSkill = { path: string; name: string; description: string };
type LibraryResponse = { skills?: Skill[]; nextPageToken?: string };
type ScanResult = { path: string; skills: ProjectSkill[] };
type Version = { id: string; version: string; commitSha?: string; sourceRef?: string; notes?: string };

function message(reason: unknown, fallback: string) {
  return reason instanceof Error && reason.message ? reason.message : fallback;
}

export default function HomePage() {
  const [skills, setSkills] = useState<Skill[]>([]);
  const [nextPageToken, setNextPageToken] = useState("");
  const [libraryLoading, setLibraryLoading] = useState(true);
  const [libraryError, setLibraryError] = useState("");
  const [keyword, setKeyword] = useState("");
  const [librarySelection, setLibrarySelection] = useState("");
  const [versions, setVersions] = useState<Version[]>([]);
  const [versionsError, setVersionsError] = useState("");
  const [versionsLoading, setVersionsLoading] = useState(false);
  const [projectPath, setProjectPath] = useState("");
  const [scanResult, setScanResult] = useState<ScanResult | null>(null);
  const [scanError, setScanError] = useState("");
  const [scanning, setScanning] = useState(false);
  const [selectedPath, setSelectedPath] = useState("");
  const [copyMessage, setCopyMessage] = useState("");

  const loadLibrary = useCallback(async (token = "", signal?: AbortSignal) => {
    setLibraryLoading(true);
    setLibraryError("");
    try {
      const result = await listSkills<LibraryResponse>({ pageSize: 100, ...(token ? { pageToken: token } : {}) }, { signal });
      if (signal?.aborted) return;
      setSkills((previous) => token ? [...previous, ...(result.skills ?? [])] : result.skills ?? []);
      setNextPageToken(result.nextPageToken ?? "");
    } catch (reason) {
      if (!signal?.aborted) setLibraryError(message(reason, "无法连接后端服务"));
    } finally {
      if (!signal?.aborted) setLibraryLoading(false);
    }
  }, []);

  useEffect(() => {
    const controller = new AbortController();
    void loadLibrary("", controller.signal);
    return () => controller.abort();
  }, [loadLibrary]);

  useEffect(() => {
    if (!librarySelection) return;
    const controller = new AbortController();
    setVersions([]);
    setVersionsError("");
    setVersionsLoading(true);
    listVersions<{ versions?: Version[] }>({ skillId: librarySelection }, { signal: controller.signal })
      .then((result) => setVersions(result.versions ?? []))
      .catch((reason: unknown) => { if (!controller.signal.aborted) setVersionsError(message(reason, "无法加载版本")); })
      .finally(() => { if (!controller.signal.aborted) setVersionsLoading(false); });
    return () => controller.abort();
  }, [librarySelection]);

  async function handleScan(event: FormEvent<HTMLFormElement>) {
    event.preventDefault();
    const path = projectPath.trim();
    if (!path || scanning) return;
    setScanning(true);
    setScanError("");
    setScanResult(null);
    setSelectedPath("");
    setCopyMessage("");
    try {
      const result = await scanProject<{ skills?: ProjectSkill[] }>({ projectPath: path });
      const found = result.skills ?? [];
      setScanResult({ path, skills: found });
      setSelectedPath(found[0]?.path ?? "");
    } catch (reason) {
      const detail = message(reason, "无法连接后端服务");
      setScanError(detail === "bad request" ? "路径无效或无法访问" : detail === "Failed to fetch" ? "无法连接后端服务" : detail);
    } finally {
      setScanning(false);
    }
  }

  async function copyPath(path: string) {
    try {
      await navigator.clipboard.writeText(path);
      setCopyMessage("路径已复制");
    } catch {
      setCopyMessage("复制失败，请手动选择路径");
    }
  }

  const query = keyword.trim().toLocaleLowerCase();
  const visibleSkills = skills.filter((skill) => `${skill.name} ${skill.description} ${skill.category ?? ""}`.toLocaleLowerCase().includes(query));
  const selected = scanResult?.skills.find((skill) => skill.path === selectedPath);
  const selectedFullPath = scanResult && selected
    ? selected.path === "." ? scanResult.path : `${scanResult.path.replace(/\/+$/, "")}/${selected.path}`
    : "";

  return (
    <main className="skillhub-shell">
      <header className="skillhub-heading">
        <div>
          <p className="skillhub-kicker">SKILL WORKSPACE</p>
          <h1>Skill 库</h1>
          <p>整理你的技能，也从项目中发现可复用的工作流。</p>
        </div>
        <a className="skillhub-heading-link" href="#project-scan">扫描项目 <span aria-hidden="true">↗</span></a>
      </header>

      <div className="skillhub-workspace">
        <section className="skillhub-section skillhub-library" aria-labelledby="library-title">
          <div className="skillhub-section-heading">
            <div><h2 id="library-title">管理库</h2><p>浏览已收录的 Skill</p></div>
            <span className="skillhub-count">已加载 {skills.length}</span>
          </div>
          <label className="skillhub-search">
            <span className="sr-only">搜索已加载的 Skill</span>
            <span aria-hidden="true">⌕</span>
            <input value={keyword} onChange={(event) => setKeyword(event.target.value)} placeholder="搜索名称、描述或分类" />
          </label>
          {nextPageToken && <p className="skillhub-hint">搜索范围为已加载的 Skill，可继续加载更多。</p>}
          {libraryError && <div className="skillhub-alert" role="alert"><span>加载失败：{libraryError}</span><button type="button" className="secondary" onClick={() => void loadLibrary(skills.length ? nextPageToken : "")} disabled={libraryLoading}>重试</button></div>}
          {libraryLoading && skills.length === 0 ? <p className="skillhub-empty" role="status">正在加载 Skill…</p> : !libraryError && skills.length === 0 ? <div className="skillhub-empty"><strong>管理库还是空的</strong><p>先扫描一个项目，查看其中有哪些 Skill。</p><a href="#project-scan">前往扫描项目 →</a></div> : visibleSkills.length === 0 && !libraryError ? <div className="skillhub-empty"><strong>没有找到匹配项</strong><p>试试其他关键词，或清除搜索条件。</p><button className="skillhub-text-button" type="button" onClick={() => setKeyword("")}>清除搜索</button></div> : (
            <ul className="skillhub-list">{visibleSkills.map((skill) => <li key={skill.id}><div className="skillhub-list-top"><button type="button" className="skillhub-library-item" aria-expanded={librarySelection === skill.id} onClick={() => setLibrarySelection(librarySelection === skill.id ? "" : skill.id)}>{skill.name}</button>{skill.category && <span className="skillhub-tag">{skill.category}</span>}</div><p>{skill.description}</p>{librarySelection === skill.id && <div className="library-versions">{skill.repositoryUrl && <p className="library-origin">来源：{skill.repositoryUrl}{skill.repositoryPath ? ` / ${skill.repositoryPath}` : ""}</p>}<strong>版本</strong>{versionsLoading ? <p>加载中…</p> : versionsError ? <p role="alert">加载失败：{versionsError}</p> : versions.length ? <ul>{versions.map((version) => <li key={version.id}><span>{version.version}</span>{version.commitSha && <code title={version.commitSha}>{version.commitSha.slice(0, 12)}</code>}{version.notes && <span>{version.notes}</span>}</li>)}</ul> : <p>暂无版本</p>}</div>}</li>)}</ul>
          )}
          {nextPageToken && !libraryError && <button className="secondary skillhub-more" type="button" disabled={libraryLoading} onClick={() => void loadLibrary(nextPageToken)}>{libraryLoading ? "加载中…" : "加载更多"}</button>}
        </section>

        <section id="project-scan" className="skillhub-section skillhub-project" aria-labelledby="project-title">
          <div className="skillhub-section-heading"><div><h2 id="project-title">发现项目 Skill</h2><p>扫描本机项目中的 Skill 目录</p></div><span className="skillhub-local">本地</span></div>
          <form className="skillhub-form" onSubmit={handleScan}>
            <label htmlFor="project-path">项目或 Skill 目录的绝对路径</label>
            <div className="skillhub-form-row"><input id="project-path" value={projectPath} onChange={(event) => setProjectPath(event.target.value)} placeholder="/Users/you/project" required /><button type="submit" disabled={scanning}>{scanning ? "扫描中…" : "扫描项目"}</button></div>
            <p className="skillhub-hint">支持项目根目录、skills 集合目录或单个 Skill 目录，不会修改文件。</p>
          </form>
          {scanError && <div className="skillhub-alert" role="alert">扫描失败：{scanError}。请检查路径和后端服务。</div>}
          {scanResult && <div className="skillhub-results" aria-live="polite">
            <div className="skillhub-results-heading"><div><strong>扫描结果</strong><p title={scanResult.path}>{scanResult.path}</p></div><span className="skillhub-count">{scanResult.skills.length} 个 Skill</span></div>
            {scanResult.skills.length === 0 ? <p className="skillhub-empty">未发现 Skill。请确认项目路径和目录位置。</p> : <div className="skillhub-scan-content">
              <ul className="skillhub-scan-list">{scanResult.skills.map((skill) => <li key={skill.path}><button type="button" className={selectedPath === skill.path ? "is-selected" : ""} aria-pressed={selectedPath === skill.path} onClick={() => { setSelectedPath(skill.path); setCopyMessage(""); }}><strong>{skill.name}</strong><span>{skill.path === "." ? "当前目录" : skill.path}</span></button></li>)}</ul>
              {selected && <div className="skillhub-detail"><span className="skillhub-detail-label">SKILL.MD</span><h3>{selected.name}</h3><p>{selected.description}</p><div className="skillhub-path"><span>{selectedFullPath}</span><button className="skillhub-text-button" type="button" onClick={() => void copyPath(selectedFullPath)}>复制路径</button></div><p className="skillhub-hint">尚未关联管理库，同步状态暂不可用。</p><span role="status" className="skillhub-hint">{copyMessage}</span></div>}
            </div>}
            {scanResult.skills.length > 0 && <IntakePanel key={scanResult.path} basePath={scanResult.path} candidates={scanResult.skills} onComplete={() => void loadLibrary()} />}
          </div>}
          <GitHubIntake onComplete={() => void loadLibrary()} />
        </section>
      </div>
    </main>
  );
}
