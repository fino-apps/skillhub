import { Fragment, lazy, Suspense, useCallback, useEffect, useRef, useState, type ReactNode } from "react";
import { getSkill, listSkills, listVersions } from "@/generated/api-client";
import GitHubIntake from "./github-intake";
import InstallCommand from "./install-command";
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
  const [installTarget, setInstallTarget] = useState<{ skillId: string; versionId: string } | null>(null);

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

  const activeSkill = skills.find((skill) => skill.id === selected);
  const query = keyword.trim().toLocaleLowerCase();
  const visible = skills.filter((skill) => `${skill.name} ${skill.description} ${skill.category ?? ""} ${skill.tags?.join(" ") ?? ""}`.toLocaleLowerCase().includes(query));

  return <main className="skillhub-shell">
    <header className="skillhub-heading">
      <h1 id="skill-library-title">Skill 库</h1>
      <div className="library-actions">
        <button type="button" className="secondary" onClick={() => setAction("compare")}>比对 Skill</button>
        <button type="button" onClick={() => setAction("add")}>添加 Skill</button>
      </div>
    </header>
    <label className="skillhub-search"><span aria-hidden="true">⌕</span><span className="sr-only">搜索 Skill</span><input value={keyword} onChange={(event) => setKeyword(event.target.value)} placeholder="搜索 Skill" /></label>
    {nextPageToken && <p className="skillhub-hint">搜索范围为已加载的 Skill。</p>}
    {error && <div className="skillhub-alert" role="alert"><span>加载失败：{error}</span><button type="button" className="secondary" onClick={() => void loadLibrary(skills.length ? nextPageToken : "")} disabled={loading}>重试</button></div>}
    {loading && !skills.length ? <p className="skillhub-empty" role="status">正在加载…</p>
      : !error && !skills.length ? <p className="skillhub-empty">还没有 Skill。<button type="button" className="skillhub-text-button" onClick={() => setAction("add")}>添加 Skill</button></p>
      : !visible.length && !error ? <p className="skillhub-empty">没有匹配的 Skill。<button type="button" className="skillhub-text-button" onClick={() => setKeyword("")}>清除搜索</button></p>
      : visible.length > 0 && <table className="skillhub-table" aria-labelledby="skill-library-title">
        <thead>
          <tr><th scope="col">Skill</th><th scope="col">用途</th><th scope="col">来源</th></tr>
        </thead>
        <tbody>{visible.map((skill) => {
          const expanded = selected === skill.id;
          return <Fragment key={skill.id}>
            <tr className="skillhub-entry">
              <th scope="row">
                <button
                  type="button"
                  className="skillhub-library-item"
                  aria-expanded={expanded}
                  onClick={() => setSelected(expanded ? "" : skill.id)}
                >
                  {skill.name}
                </button>
              </th>
              <td>{skill.description}</td>
              <td>
                {skill.localPath ? <>本地 · {skill.localPath.split("/").at(-1)}</>
                  : skill.repositoryUrl && <a href={skill.repositoryUrl} target="_blank" rel="noreferrer">
                    {skill.repositoryUrl.replace(/^https:\/\/github\.com\//i, "")}
                  </a>}
              </td>
            </tr>
            {expanded && <tr className="skillhub-details">
              <td colSpan={3}>
                {skill.localPath && <p>原目录 <code>{skill.localPath}</code></p>}
                {skill.category && <p>分类 · {skill.category}</p>}
                {!!skill.tags?.length && <p>标签 · {skill.tags.join("、")}</p>}
                {skill.notes && <p className="library-notes">{skill.notes}</p>}
                <div className="library-actions">
                  <button type="button" className="skillhub-text-button" onClick={() => setAction("metadata")}>编辑整理信息</button>
                  {skill.repositoryUrl && versions[0]?.commitSha && <a
                    className="skillhub-text-button"
                    href={`${skill.repositoryUrl}/tree/${versions[0].commitSha}${skill.repositoryPath ? `/${skill.repositoryPath}` : ""}`}
                    target="_blank"
                    rel="noreferrer"
                  >查看 GitHub 收录版本</a>}
                </div>
                <div className="library-version-heading">
                  <h2>版本</h2>
                  <button type="button" className="skillhub-text-button" disabled={versionsLoading || !!versionsError} onClick={() => setAction("version")}>
                    {skill.localPath ? "保存新版本" : "检查新版本"}
                  </button>
                  <button type="button" className="skillhub-text-button" disabled={versionsLoading || !versions.length} onClick={() => setAction("compare-versions")}>比对版本</button>
                </div>
                {versionsError && <p className="intake-error" role="alert">加载失败：{versionsError}</p>}
                {versions.length > 0 && <ul className="library-versions">{versions.map((version) => <li key={version.id}>
                  <span>{versionLabel(version)}</span>
                  <code title={version.commitSha ?? version.contentDigest}>{(version.commitSha ?? version.contentDigest)?.slice(0, 12)}</code>
                  {version.notes && <span>{version.notes}</span>}
                  <button type="button" className="skillhub-text-button" onClick={() => setInstallTarget({ skillId: skill.id, versionId: version.id })}>安装</button>
                </li>)}</ul>}
                {versionsLoading && <p role="status">加载中…</p>}
                {!versionsLoading && !versionsError && !versions.length && <p>暂无版本</p>}
                {moreVersions && <button type="button" className="skillhub-text-button" disabled={versionsLoading} onClick={() => void loadVersions(skill.id, moreVersions, versionsRequest.current?.signal)}>更多版本</button>}
              </td>
            </tr>}
          </Fragment>;
        })}</tbody>
      </table>}
    {nextPageToken && !error && <button className="secondary skillhub-more" type="button" disabled={loading} onClick={() => void loadLibrary(nextPageToken)}>{loading ? "加载中…" : "加载更多"}</button>}
    {installTarget && <ActionDialog title="Codex 安装命令" compact onClose={() => setInstallTarget(null)}>
      <InstallCommand {...installTarget} />
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
