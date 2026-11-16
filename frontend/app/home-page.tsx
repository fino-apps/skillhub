import { lazy, Suspense, useEffect, useRef, useState } from "react";
import { Link, useSearchParams } from "react-router-dom";
import { deleteSkill, listFacets, listSkills, repairVersion } from "@/generated/api-client";
import GitHubIntake from "./github-intake";
import LocalIntake, { ZipIntake } from "./local-intake";
import SkillMetadata, { BatchMetadata } from "./skill-metadata";
import { archiveURL, fetchAllSkills, libraryNavigation, message, sourceLabel, sourceNames, type Facets, type ImportResult, type LibraryPage, type Skill } from "./library-api";
import { Dialog, ExportLocation, Failure, Menu, OperationFeedback, useSkillOperation } from "./skill-ui";
import { shortVersion, versionDigest } from "./skill-version";

import "./skillhub.css";

const ComparePanel = lazy(() => import("./compare-panel"));
const VersionPanel = lazy(() => import("./version-panel"));
const OrganizePanel = lazy(() => import("./organize-panel"));

type Panel =
  | { kind: "folder" | "github" | "zip" | "batch" | "ai" }
  | { kind: "metadata"; skill: Skill }
  | { kind: "new-version"; skill: Skill; path: string }
  | { kind: "export"; skill: Skill; versionId: string }
  | { kind: "compare"; skills: Skill[] };

function Filters({ params, facets, onChange }: {
  params: URLSearchParams; facets: Facets; onChange: (next: URLSearchParams) => void;
}) {
  const source = params.get("source") ?? "";
  const tags = params.getAll("tag");
  const count = Number(!!source) + tags.length;

  function set(name: string, value: string) {
    const next = new URLSearchParams(params);
    if (value) next.set(name, value);
    else next.delete(name);
    onChange(next);
  }

  return <div className="library-filterbar">
    <label className="skillhub-search">
      <span aria-hidden="true">⌕</span><span className="sr-only">搜索 Skill</span>
      <input type="search" placeholder="搜索名称、用途或标签" value={params.get("q") ?? ""} onChange={(event) => set("q", event.target.value)} />
    </label>
    <label><span className="sr-only">分类</span>
      <select value={params.get("category") ?? ""} onChange={(event) => set("category", event.target.value)}>
        <option value="">全部分类</option><option value="~">未分类</option>
        {facets.categories?.map((category) => <option key={category}>{category}</option>)}
      </select>
    </label>
    <details className="skill-filter">
      <summary>筛选{count ? ` ${count}` : ""}</summary>
      <div className="filter-panel skill-fields">
        <label>来源<select value={source} onChange={(event) => set("source", event.target.value)}>
          <option value="">全部来源</option>
          {facets.sources?.map((value) => <option key={value} value={value}>{sourceNames[value] || value}</option>)}
        </select></label>
        {!!facets.tags?.length && <fieldset className="skill-fields">
          <legend>标签</legend>
          {facets.tags.map((tag) => <label key={tag} className="github-checkbox">
            <input type="checkbox" checked={tags.includes(tag)} onChange={(event) => {
              const next = new URLSearchParams(params);
              next.delete("tag");
              for (const value of event.target.checked ? [...tags, tag] : tags.filter((value) => value !== tag)) {
                next.append("tag", value);
              }
              onChange(next);
            }} />{tag}
          </label>)}
        </fieldset>}
        {count > 0 && <button className="secondary" type="button" onClick={() => {
          const next = new URLSearchParams(params);
          next.delete("source");
          next.delete("tag");
          onChange(next);
        }}>清除筛选</button>}
      </div>
    </details>
  </div>;
}

export default function HomePage() {
  const [params, setParams] = useSearchParams();
  const scope = params.toString();
  const initial = useRef(libraryNavigation.returning?.scope === scope ? libraryNavigation.returning : null);
  const request = useRef<AbortController | null>(null);
  const [skills, setSkills] = useState(initial.current?.skills ?? []);
  const [token, setToken] = useState(initial.current?.token ?? "");
  const [selected, setSelected] = useState<string[]>([]);
  const [facets, setFacets] = useState<Facets>({});
  const [panel, setPanel] = useState<Panel | null>(null);
  const [refresh, setRefresh] = useState(0);
  const [loading, setLoading] = useState(!initial.current);
  const [error, setError] = useState("");
  const operation = useSkillOperation();

  useEffect(() => {
    const controller = new AbortController();
    request.current = controller;
    const saved = refresh === 0 && initial.current?.scope === scope ? initial.current : null;
    let frame = 0;
    const delay = setTimeout(async () => {
      if (!saved || saved.revision !== libraryNavigation.revision) {
        await load("", controller, saved?.skills.length ?? 0);
      }
      if (saved && !controller.signal.aborted) {
        frame = requestAnimationFrame(() => {
          initial.current = null;
          libraryNavigation.returning = null;
          const link = document.getElementById(`skill-link-${saved.skillId}`);
          if (link) window.scrollTo(0, saved.scroll);
          (link ?? document.getElementById("skill-library-heading"))?.focus({ preventScroll: !!link });
        });
      }
    }, saved ? 0 : 140);
    return () => { clearTimeout(delay); cancelAnimationFrame(frame); controller.abort(); };
  }, [scope, refresh]);

  useEffect(() => {
    const controller = new AbortController();
    void listFacets<Facets>({}, { signal: controller.signal }).then(setFacets, () => {});
    return () => controller.abort();
  }, [refresh]);

  async function load(pageToken: string, controller = new AbortController(), restoreCount = 0) {
    if (pageToken) {
      request.current?.abort();
      request.current = controller;
    }
    setLoading(true);
    setError("");
    const category = params.get("category") ?? "";
    const filter = {
      keyword: params.get("q") ?? "", category: category === "~" ? "" : category,
      source: params.get("source") ?? "", tags: params.getAll("tag"), uncategorized: category === "~", pageSize: 40,
    };
    try {
      const items: Skill[] = [];
      let next = pageToken;
      do {
        const result = await listSkills<LibraryPage>({ ...filter, pageToken: next }, { signal: controller.signal });
        items.push(...(result.skills ?? []));
        next = result.nextPageToken ?? "";
      } while (next && items.length < restoreCount && !controller.signal.aborted);
      if (controller.signal.aborted) return;
      setSkills((previous) => pageToken ? [...previous, ...items.filter(({ id }) => !previous.some((skill) => skill.id === id))] : items);
      setToken(next);
    } catch (err) {
      if (!controller.signal.aborted) setError(message(err));
    } finally {
      if (!controller.signal.aborted) setLoading(false);
    }
  }

  function updateFilters(next: URLSearchParams) {
    setSelected([]);
    setParams(next, { replace: true });
  }

  function reload() {
    setSelected([]);
    setRefresh((value) => value + 1);
  }

  function imported(result: ImportResult) {
    reload();
    if (result.failures?.length) return;
    setPanel(null);
    operation.setNotice(`新增 ${result.addedCount ?? 0} 项，已有 ${result.existingCount ?? 0} 项。`);
  }

  async function compare() {
    await operation.run("compare", async () => setPanel({ kind: "compare", skills: await fetchAllSkills() }));
  }

  function remove(skill: Skill) {
    if (!window.confirm(`删除「${skill.name}」及其已保存版本？`)) return;
    void operation.run(skill.id, async () => {
      await deleteSkill({ skillId: skill.id });
      reload();
    });
  }

  const uncategorized = params.get("category") === "~";
  const selectable = skills.slice(0, 100);
  const allSelected = selectable.length > 0 && selectable.every(({ id }) => selected.includes(id));
  const library = scope ? `/?${scope}` : "/";
  const aiSkills = selected.length
    ? skills.filter(({ id }) => selected.includes(id))
    : params.has("q") || params.has("source") || params.has("tag") ? skills.slice(0, 100) : [];
  const editComplete = () => { setPanel(null); reload(); };

  return <main className="skillhub-shell">
    <div className="skillhub-heading">
      <h1 id="skill-library-heading" tabIndex={-1}>技能库</h1>
      <div className="library-actions">
        <button type="button" onClick={() => setPanel({ kind: "folder" })}>导入文件夹</button>
        <button type="button" className="secondary" onClick={() => setPanel({ kind: "github" })}>导入 GitHub</button>
        <Menu label="技能库更多操作">
          <button type="button" onClick={() => setPanel({ kind: "zip" })}>导入 ZIP</button>
          <button type="button" disabled={!!operation.busy} onClick={() => void compare()}>比对 Skill</button>
        </Menu>
      </div>
    </div>
    <Filters params={params} facets={facets} onChange={updateFilters} />
    <div className="library-feedback"><OperationFeedback {...operation} /></div>
    {(selected.length > 0 || (uncategorized && skills.length > 0)) && <div className="library-actions library-selection">
      {selected.length > 0 && <>
        <span>已选 {selected.length} 项</span>
        <button type="button" className="secondary" onClick={() => setPanel({ kind: "batch" })}>批量编辑</button>
      </>}
      <button type="button" className="secondary" onClick={() => setPanel({ kind: "ai" })}>AI 整理</button>
    </div>}
    {error && <div className="library-actions"><Failure error={error} /><button type="button" className="secondary" onClick={reload}>重试</button></div>}
    {loading && skills.length === 0 && <p className="skillhub-empty" role="status">加载中…</p>}
    {!loading && !error && skills.length === 0 && <div className="skillhub-empty">
      <p>{scope ? "没有匹配的 Skill" : "还没有 Skill"}</p>
      {scope ? <button type="button" className="secondary" onClick={() => updateFilters(new URLSearchParams())}>清除筛选</button>
        : <button type="button" onClick={() => setPanel({ kind: "folder" })}>导入文件夹</button>}
    </div>}
    {skills.length > 0 && <table className="skillhub-table library-table" aria-busy={loading}>
      <thead><tr>
        <th scope="col"><input type="checkbox" aria-label="选择当前已加载条目" checked={allSelected}
          onChange={() => setSelected(allSelected ? [] : selectable.map(({ id }) => id))} /></th>
        <th scope="col">Skill / 用途</th><th scope="col">来源</th><th scope="col">分类 / 标签</th><th scope="col">操作</th>
      </tr></thead>
      <tbody>{skills.map((skill) => {
        const version = skill.defaultVersion;
        const tags = skill.tags ?? [];
        const purpose = skill.purpose || skill.description || "";
        return <tr key={skill.id} className="skillhub-entry">
          <td><input type="checkbox" aria-label={`选择 ${skill.name}`} checked={selected.includes(skill.id)}
            disabled={selected.length >= 100 && !selected.includes(skill.id)}
            onChange={() => setSelected((ids) => ids.includes(skill.id) ? ids.filter((id) => id !== skill.id) : [...ids, skill.id])} /></td>
          <td>
            <Link id={`skill-link-${skill.id}`} className="library-name" to={`/skills/${skill.id}`} state={{ library }}
              onClick={() => { libraryNavigation.returning = { scope, skills, token, scroll: window.scrollY, skillId: skill.id, revision: libraryNavigation.revision }; }}>{skill.name}</Link>
            {purpose && <span className="library-purpose" title={purpose}>{purpose}</span>}
            <span className="library-mobile-source">{sourceLabel(skill)}{skill.category ? ` · ${skill.category}` : ""}</span>
          </td>
          <td className="library-source">{sourceLabel(skill)}</td>
          <td className="library-classification">{[skill.category, ...tags.slice(0, 2)].filter(Boolean).join(" · ") || "—"}{tags.length > 2 && ` +${tags.length - 2}`}</td>
          <td className="library-row-actions">
            {version?.available && <code title={versionDigest(version)}>{shortVersion(version)}</code>}
            {version && (version.available
              ? <button type="button" className="skillhub-text-button" disabled={!!operation.busy}
                onClick={() => void operation.save(skill.id, version.id)}>{operation.busy === version.id ? "保存中…" : "保存到本机"}</button>
              : <button type="button" className="skillhub-text-button" disabled={!!operation.busy} onClick={() => void operation.run(version.id, async () => {
                await repairVersion({ skillId: skill.id, versionId: version.id });
                operation.setNotice("版本已补齐，可以保存到本机。");
                reload();
              })}>补齐版本</button>)}
            <Menu label={`${skill.name} 更多操作`}>
              {version?.available && <>
                <button type="button" onClick={() => setPanel({ kind: "export", skill, versionId: version.id })}>另存到…</button>
                <a href={archiveURL(skill.id, version.id)}>下载 ZIP</a>
              </>}
              <button type="button" onClick={() => setPanel({ kind: "metadata", skill })}>编辑整理信息</button>
              <button type="button" className="skill-danger" onClick={() => remove(skill)}>删除</button>
            </Menu>
          </td>
        </tr>;
      })}</tbody>
    </table>}
    {token && <button className="secondary skillhub-more" type="button" disabled={loading} onClick={() => void load(token)}>{loading ? "加载中…" : "加载更多"}</button>}

    {panel?.kind === "folder" && <Dialog title="导入文件夹" compact onClose={() => setPanel(null)}>
      <LocalIntake onComplete={imported} onCancel={() => setPanel(null)}
        onVersion={(skill, path) => setPanel({ kind: "new-version", skill, path })} />
    </Dialog>}
    {panel?.kind === "github" && <Dialog title="导入 GitHub" compact onClose={() => setPanel(null)}><GitHubIntake onComplete={imported} /></Dialog>}
    {panel?.kind === "zip" && <Dialog title="导入 ZIP" compact onClose={() => setPanel(null)}><ZipIntake onComplete={imported} /></Dialog>}
    {panel?.kind === "metadata" && <Dialog title="编辑整理信息" compact onClose={() => setPanel(null)}>
      <SkillMetadata skill={panel.skill} categories={facets.categories ?? []} onSaved={editComplete} />
    </Dialog>}
    {panel?.kind === "new-version" && <Dialog title="保存新版本" onClose={() => setPanel(null)}>
      <Suspense fallback={<p>加载中…</p>}><VersionPanel skill={panel.skill} from="folder" path={panel.path} onSaved={editComplete} /></Suspense>
    </Dialog>}
    {panel?.kind === "batch" && <Dialog title="批量编辑" compact onClose={() => setPanel(null)}>
      <BatchMetadata ids={selected} categories={facets.categories ?? []} onSaved={editComplete} />
    </Dialog>}
    {panel?.kind === "ai" && <Dialog title="AI 整理" onClose={() => setPanel(null)}>
      {facets.aiEnabled
        ? <Suspense fallback={<p>加载中…</p>}><OrganizePanel skills={aiSkills}
          uncategorized={uncategorized} categories={facets.categories ?? []} onSaved={editComplete} /></Suspense>
        : <p>启动本机 Ollama，并在启动 SkillHub 时设置 <code>SKILLHUB_OLLAMA_MODEL</code>。</p>}
    </Dialog>}
    {panel?.kind === "compare" && <Dialog title="比对 Skill" onClose={() => setPanel(null)}>
      <Suspense fallback={<p>加载中…</p>}><ComparePanel skills={panel.skills} /></Suspense>
    </Dialog>}
    {panel?.kind === "export" && <Dialog title="另存到本机" compact onClose={() => setPanel(null)}>
      <ExportLocation busy={!!operation.busy} onSave={(directory) => void operation.save(panel.skill.id, panel.versionId, directory === undefined, directory)} />
      <OperationFeedback {...operation} />
    </Dialog>}
  </main>;
}
