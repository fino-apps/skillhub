import { lazy, Suspense, useCallback, useEffect, useRef, useState, type ReactNode } from "react";
import { getInstallCommand, getSkill, listSkills, listVersions } from "@/generated/api-client";
import GitHubIntake from "./github-intake";
import LocalIntake from "./local-intake";
import { versionLabel, type SkillVersion } from "./skill-version";
import SkillMetadata from "./skill-metadata";

import "./skillhub.css";

const ComparePanel = lazy(() => import("./compare-panel"));
const VersionPanel = lazy(() => import("./version-panel"));

type Skill = { id: string; name: string; description: string; category?: string; notes?: string; tags?: string[]; repositoryUrl?: string; repositoryPath?: string; localPath?: string };
type LibraryResponse = { skills?: Skill[]; nextPageToken?: string };

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
  const [source, setSource] = useState<"local" | "github">("local");
  return <>
    <div className="skill-source-toggle" role="group" aria-label="收录来源">
      <button type="button" className="secondary" aria-pressed={source === "local"} onClick={() => setSource("local")}>本地目录</button>
      <button type="button" className="secondary" aria-pressed={source === "github"} onClick={() => setSource("github")}>GitHub</button>
    </div>
    <div hidden={source !== "local"}><LocalIntake onComplete={onComplete} /></div>
    <div hidden={source !== "github"}><GitHubIntake onComplete={onComplete} /></div>
  </>;
}

export default function HomePage() {
  const [skills, setSkills] = useState<Skill[]>([]);
  const [nextPageToken, setNextPageToken] = useState("");
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState("");
  const [keyword, setKeyword] = useState("");
  const [selected, setSelected] = useState("");
  const [versions, setVersions] = useState<SkillVersion[]>([]);
  const [versionsError, setVersionsError] = useState("");
  const [versionsLoading, setVersionsLoading] = useState(false);
  const [action, setAction] = useState<"add" | "compare" | "compare-versions" | "version" | "metadata" | null>(null);
  const [versionRevision, setVersionRevision] = useState(0);
  const [moreVersions, setMoreVersions] = useState("");
  const versionsRequest = useRef<AbortController | null>(null);
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

  const loadVersions = useCallback(async (skillId: string, token = "", signal?: AbortSignal) => {
    setVersionsLoading(true);
    setVersionsError("");
    try {
      const result = await listVersions<{ versions?: SkillVersion[]; nextPageToken?: string }>({
        skillId, pageSize: 30, ...(token ? { pageToken: token } : {}),
      }, { signal });
      if (!signal?.aborted) {
        setVersions((current) => token ? [...current, ...(result.versions ?? [])] : result.versions ?? []);
        setMoreVersions(result.nextPageToken ?? "");
      }
    } catch (reason) {
      if (!signal?.aborted) setVersionsError(message(reason, "无法加载版本"));
    } finally {
      if (!signal?.aborted) setVersionsLoading(false);
    }
  }, []);

  useEffect(() => {
    if (!selected) return;
    const controller = new AbortController();
    versionsRequest.current = controller;
    setVersions([]);
    setMoreVersions("");
    void loadVersions(selected, "", controller.signal);
    return () => controller.abort();
  }, [selected, versionRevision, loadVersions]);

  async function prepareCommand(skillId: string, versionId: string, global = false) {
    if (generating) return;
    setGenerating(versionId);
    setCommandError("");
    try {
      const result = await getInstallCommand<{ command: string }>({ skillId, versionId, global });
      setCopyStatus("");
      setCommand(result.command);
    } catch (reason) {
      setCommandError(message(reason, "请稍后重试"));
    } finally {
      setGenerating("");
    }
  }

  function replaceSkill(updated: Skill) {
    setSkills((current) => current.map((skill) => skill.id === updated.id ? updated : skill));
  }

  async function refreshVersionSkill(skillId: string) {
    setAction(null);
    setVersionRevision((value) => value + 1);
    try {
      const result = await getSkill<{ skill: Skill }>({ skillId });
      replaceSkill(result.skill);
    } catch {
      void loadLibrary();
    }
  }

  async function copy() {
    try {
      await navigator.clipboard.writeText(command);
      setCopyStatus("已复制");
    } catch {
      setCopyStatus("复制失败，请手动选择命令");
    }
  }

  const activeSkill = skills.find((skill) => skill.id === selected);
  const query = keyword.trim().toLocaleLowerCase();
  const visible = skills.filter((skill) => `${skill.name} ${skill.description} ${skill.category ?? ""} ${skill.tags?.join(" ") ?? ""}`.toLocaleLowerCase().includes(query));

  return <main className="skillhub-shell">
    <header className="skillhub-heading"><div><h1>Skill 库</h1><span className="skillhub-count">{skills.length} 个已加载</span></div><div className="library-actions"><button type="button" className="secondary" onClick={() => setAction("compare")}>比对 Skill</button><button type="button" onClick={() => setAction("add")}>添加 Skill</button></div></header>
    <label className="skillhub-search"><span aria-hidden="true">⌕</span><span className="sr-only">搜索 Skill</span><input value={keyword} onChange={(event) => setKeyword(event.target.value)} placeholder="搜索 Skill" /></label>
    {nextPageToken && <p className="skillhub-hint">搜索范围为已加载的 Skill。</p>}
    {error && <div className="skillhub-alert" role="alert"><span>加载失败：{error}</span><button type="button" className="secondary" onClick={() => void loadLibrary(skills.length ? nextPageToken : "")} disabled={loading}>重试</button></div>}
    {loading && !skills.length ? <p className="skillhub-empty" role="status">正在加载…</p>
      : !error && !skills.length ? <p className="skillhub-empty">还没有 Skill。<button type="button" className="skillhub-text-button" onClick={() => setAction("add")}>添加 Skill</button></p>
      : !visible.length && !error ? <p className="skillhub-empty">没有匹配的 Skill。<button type="button" className="skillhub-text-button" onClick={() => setKeyword("")}>清除搜索</button></p>
      : <ul className="skillhub-list">{visible.map((skill) => <li key={skill.id}>
        <div className="skillhub-list-top">
          <button type="button" className="skillhub-library-item" aria-expanded={selected === skill.id} onClick={() => setSelected(selected === skill.id ? "" : skill.id)}>{skill.name}</button>
          <div className="library-tags">
            <span className="skillhub-tag">{skill.localPath ? "本地" : "GitHub"}</span>
            {skill.category && <span className="skillhub-tag">{skill.category}</span>}
            {skill.tags?.map((tag) => <span key={tag} className="skillhub-tag">{tag}</span>)}
          </div>
        </div>
        <p>{skill.description}</p>
        {selected === skill.id && <div className="library-versions">
          <div className="library-version-heading">
            {skill.repositoryUrl && <a className="skillhub-text-button" href={versions[0]?.commitSha
              ? `${skill.repositoryUrl}/tree/${versions[0].commitSha}${skill.repositoryPath ? `/${skill.repositoryPath}` : ""}`
              : skill.repositoryUrl} target="_blank" rel="noreferrer">在 GitHub 打开</a>}
            <button type="button" className="skillhub-text-button" disabled={versionsLoading || !!versionsError} onClick={() => setAction("version")}>{skill.localPath ? "保存新版本" : "检查新版本"}</button>
            <button type="button" className="skillhub-text-button" disabled={versionsLoading || !versions.length} onClick={() => setAction("compare-versions")}>比对版本</button>
            <button type="button" className="skillhub-text-button" onClick={() => setAction("metadata")}>编辑整理信息</button>
          </div>
          {skill.localPath && <p className="library-notes">原目录 <code>{skill.localPath}</code></p>}
          {skill.notes && <p className="library-notes">{skill.notes}</p>}
          <div className="library-version-row">
            <span className="library-version-label">版本</span>
            {versionsError && <p role="alert">加载失败：{versionsError}</p>}
            {versions.length > 0 && <ul>{versions.map((version) => <li key={version.id}>
              <span>{versionLabel(version)}</span>
              <code title={version.commitSha ?? version.contentDigest}>{(version.commitSha ?? version.contentDigest)?.slice(0, 12)}</code>
              {version.notes && <span>{version.notes}</span>}
              <button type="button" className="skillhub-text-button" disabled={!!generating} onClick={() => void prepareCommand(skill.id, version.id)}>安装到项目</button>
              <button type="button" className="skillhub-text-button" disabled={!!generating} onClick={() => void prepareCommand(skill.id, version.id, true)}>全局安装</button>
            </li>)}</ul>}
            {versionsLoading && <p role="status">加载中…</p>}
            {!versionsLoading && !versionsError && !versions.length && <p>暂无版本</p>}
            {moreVersions && <button type="button" className="skillhub-text-button" disabled={versionsLoading} onClick={() => void loadVersions(skill.id, moreVersions, versionsRequest.current?.signal)}>更多版本</button>}
          </div>
        </div>}
      </li>)}</ul>}
    {nextPageToken && !error && <button className="secondary skillhub-more" type="button" disabled={loading} onClick={() => void loadLibrary(nextPageToken)}>{loading ? "加载中…" : "加载更多"}</button>}
    {commandError && <p className="intake-error" role="alert">无法生成命令：{commandError}</p>}
    {command && <ActionDialog title="Codex 安装命令" compact onClose={() => setCommand("")}>
      <p className="skillhub-hint copy-hint">在本机终端运行。项目级安装请先进入目标项目，重装可能替换同名 Skill。</p>
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
    {(action === "compare" || action === "compare-versions") && <ActionDialog title={action === "compare-versions" ? "比对版本" : "比对 Skill"} onClose={() => setAction(null)}><Suspense fallback={<p role="status">加载中…</p>}><ComparePanel skills={skills} skillId={action === "compare-versions" ? selected : undefined} /></Suspense></ActionDialog>}
    {action === "version" && activeSkill && <ActionDialog title={activeSkill.localPath ? "保存新版本" : "检查新版本"} onClose={() => setAction(null)}>
      <Suspense fallback={<p role="status">加载中…</p>}>
        <VersionPanel skill={activeSkill} latest={versions[0]} onSaved={() => void refreshVersionSkill(activeSkill.id)} />
      </Suspense>
    </ActionDialog>}
    {action === "metadata" && activeSkill && <ActionDialog title="编辑整理信息" compact onClose={() => setAction(null)}>
      <SkillMetadata skill={activeSkill} onSaved={(updated) => { replaceSkill(updated); setAction(null); }} />
    </ActionDialog>}
  </main>;
}
