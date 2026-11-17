import { lazy, Suspense, useEffect, useRef, useState, type ReactNode } from "react";
import { Link, useSearchParams } from "react-router-dom";
import { listFacets, listSkills, repairVersion } from "@/generated/api-client";
import GitHubIntake from "./github-intake";
import LocalIntake, { ZipIntake } from "./local-intake";
import { BatchMetadata } from "./skill-metadata";
import { fetchAllSkills, libraryNavigation, message, sourceLabel, sourceNames, type Facets, type ImportResult, type LibraryPage, type Skill } from "./library-api";
import { Dialog, Failure, Menu, OperationFeedback, Popover, useSkillOperation } from "./skill-ui";

import "./skillhub.css";

const ComparePanel = lazy(() => import("./compare-panel"));
const VersionPanel = lazy(() => import("./version-panel"));
const OrganizePanel = lazy(() => import("./organize-panel"));

type Panel =
  | { kind: "folder" | "github" | "zip" | "batch" | "ai" }
  | { kind: "new-version"; skill: Skill; path: string }
  | { kind: "compare"; skills: Skill[] };

function LibraryToolbar({ params, facets, onChange, status, children }: {
  params: URLSearchParams; facets: Facets; onChange: (next: URLSearchParams) => void; status: ReactNode; children: ReactNode;
}) {
  const source = params.get("source") ?? "";
  const tags = params.getAll("tag");
  const category = params.get("category") ?? "";

  function set(name: string, value: string) {
    const next = new URLSearchParams(params);
    if (value) next.set(name, value);
    else next.delete(name);
    onChange(next);
  }

  function setTags(values: string[]) {
    const next = new URLSearchParams(params);
    next.delete("tag");
    values.forEach((value) => next.append("tag", value));
    onChange(next);
  }

  return <>
    <div className="library-filterbar">
      <h1 id="skill-library-heading" className="library-heading" tabIndex={-1}>技能库</h1>
      <label className="skillhub-search">
        <span aria-hidden="true">⌕</span><span className="sr-only">搜索 Skill</span>
        <input type="search" name="q" autoComplete="off" placeholder="搜索名称、用途或标签…" value={params.get("q") ?? ""} onChange={(event) => set("q", event.target.value)} />
      </label>
      <div className="library-facets">
        <label><span className="sr-only">来源</span>
          <select value={source} onChange={(event) => set("source", event.target.value)}>
            <option value="">全部来源</option>
            {source && !facets.sources?.includes(source) && <option value={source}>{sourceNames[source] || source}</option>}
            {facets.sources?.map((value) => <option key={value} value={value}>{sourceNames[value] || value}</option>)}
          </select>
        </label>
        <label><span className="sr-only">分类</span>
          <select value={category} onChange={(event) => set("category", event.target.value)}>
            <option value="">全部分类</option><option value="~">未分类</option>
            {category && category !== "~" && !facets.categories?.includes(category) && <option>{category}</option>}
            {facets.categories?.map((category) => <option key={category}>{category}</option>)}
          </select>
        </label>
        {!!facets.tags?.length && <Popover className={`skill-filter${tags.length ? " has-filters" : ""}`} summary="标签">
          <div className="filter-panel skill-fields">
            <fieldset className="skill-fields filter-tags">
              <legend>标签</legend>
              {facets.tags.map((tag) => <label key={tag} className="github-checkbox">
                <input type="checkbox" checked={tags.includes(tag)} onChange={(event) =>
                  setTags(event.target.checked ? [...tags, tag] : tags.filter((value) => value !== tag))} />{tag}
              </label>)}
            </fieldset>
          </div>
        </Popover>}
      </div>
      <div className="library-filter-end">
        <div className="library-filter-status">
          {status}
          {params.size > 0 && <button type="button" className="skillhub-text-button"
            onClick={() => onChange(new URLSearchParams())}>清除筛选</button>}
        </div>
        {children}
      </div>
    </div>
    {!!tags.length && <div className="library-tags" aria-label="已选标签" role="group">
      {tags.map((tag) => <button key={tag} type="button" className="secondary" aria-label={`移除标签 ${tag}`}
        onClick={() => setTags(tags.filter((value) => value !== tag))}>{tag} <span aria-hidden="true">×</span></button>)}
    </div>}
  </>;
}

export default function HomePage() {
  const [params, setParams] = useSearchParams();
  const scope = params.toString();
  const initial = useRef(libraryNavigation.returning?.scope === scope ? libraryNavigation.returning : null);
  const request = useRef<AbortController | null>(null);
  const selectAll = useRef<HTMLInputElement>(null);
  const [results, setResults] = useState<{ scope: string; skills: Skill[]; token: string } | null>(initial.current);
  const [selected, setSelected] = useState<string[]>([]);
  const [facets, setFacets] = useState<Facets>({});
  const [panel, setPanel] = useState<Panel | null>(null);
  const [refresh, setRefresh] = useState(0);
  const [pending, setPending] = useState(!initial.current);
  const [failure, setFailure] = useState<{ scope: string; pageToken: string; message: string } | null>(null);
  const operation = useSkillOperation();
  const current = results?.scope === scope ? results : null;
  const skills = current?.skills ?? [];
  const token = current?.token ?? "";
  const error = failure?.scope === scope ? failure : null;
  const loading = !error && (pending || !current);

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
    return () => {
      clearTimeout(delay);
      cancelAnimationFrame(frame);
      controller.abort();
      request.current?.abort();
    };
  }, [scope, refresh]);

  useEffect(() => {
    const controller = new AbortController();
    void listFacets<Facets>({}, { signal: controller.signal }).then(setFacets, () => {});
    return () => controller.abort();
  }, [refresh]);

  async function load(pageToken: string, controller = new AbortController(), restoreCount = 0) {
    if (request.current !== controller) {
      request.current?.abort();
      request.current = controller;
    }
    setPending(true);
    setFailure(null);
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
      setResults((previous) => {
        const loaded = pageToken && previous?.scope === scope ? previous.skills : [];
        const ids = new Set(loaded.map(({ id }) => id));
        return { scope, skills: [...loaded, ...items.filter(({ id }) => !ids.has(id))], token: next };
      });
    } catch (err) {
      if (!controller.signal.aborted) setFailure({ scope, pageToken, message: message(err) });
    } finally {
      if (!controller.signal.aborted) setPending(false);
    }
  }

  function updateFilters(next: URLSearchParams) {
    if (next.toString() === scope) return;
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

  const uncategorized = params.get("category") === "~";
  const selectable = skills.slice(0, 100);
  const allSelected = selectable.length > 0 && selectable.every(({ id }) => selected.includes(id));
  useEffect(() => {
    if (selectAll.current) selectAll.current.indeterminate = selected.length > 0 && !allSelected;
  }, [selected.length, allSelected, current]);
  const library = scope ? `/?${scope}` : "/";
  const aiSkills = selected.length
    ? skills.filter(({ id }) => selected.includes(id))
    : params.has("q") || params.has("source") || params.has("tag") ? skills.slice(0, 100) : [];
  const editComplete = () => { setPanel(null); reload(); };

  return <main className="skillhub-shell library-shell">
    <LibraryToolbar params={params} facets={facets} onChange={updateFilters} status={<>
      <span role="status">{!current && loading
        ? "正在查找技能…" : current && `${token ? "已加载" : "共"} ${skills.length} 项`}</span>
      {uncategorized && skills.length > 0 &&
        <button type="button" className="secondary" style={{ visibility: selected.length ? "hidden" : undefined }}
          onClick={() => setPanel({ kind: "ai" })}>AI 整理</button>}
    </>}>
      <div className="library-actions library-import-actions">
        <button type="button" onClick={() => setPanel({ kind: "folder" })}>导入文件夹</button>
        <button type="button" className="secondary" onClick={() => setPanel({ kind: "github" })}>导入 GitHub</button>
        <Menu label="技能库更多操作">
          <button type="button" onClick={() => setPanel({ kind: "zip" })}>导入 ZIP</button>
          <button type="button" disabled={!!operation.busy} onClick={() => void compare()}>比对 Skill</button>
        </Menu>
      </div>
    </LibraryToolbar>
    <div className="library-overlays">
      <OperationFeedback {...operation} floating />
      {selected.length > 0 && <div className="library-actions library-selection" role="group" aria-label="批量操作">
        <span role="status">已选 {selected.length} 项</span>
        <button type="button" className="secondary" onClick={() => setPanel({ kind: "batch" })}>批量编辑</button>
        <button type="button" className="secondary" onClick={() => setPanel({ kind: "ai" })}>AI 整理</button>
        <button type="button" className="skillhub-text-button" onClick={() => {
          setSelected([]);
          selectAll.current?.focus({ preventScroll: true });
        }}>清除选择</button>
      </div>}
    </div>
    {error && !error.pageToken && <div className="library-actions"><Failure error={error.message} /><button type="button" className="secondary" onClick={() => void load("")}>重试</button></div>}
    {!loading && !error && skills.length === 0 && <div className="skillhub-empty">
      <p>{scope ? "没有匹配的 Skill" : "还没有 Skill"}</p>
      {!scope && <button type="button" className="secondary" onClick={() => setPanel({ kind: "folder" })}>导入文件夹</button>}
    </div>}
    {skills.length > 0 && <table className="skillhub-table library-table" aria-busy={loading}>
      <thead><tr>
        <th scope="col"><input ref={selectAll} type="checkbox" aria-label={skills.length > 100 ? "选择前 100 项" : "选择当前已加载条目"} checked={allSelected}
          onChange={() => setSelected(allSelected ? [] : selectable.map(({ id }) => id))} /></th>
        <th scope="col">Skill / 用途</th><th scope="col">来源</th><th scope="col">分类 / 标签</th><th scope="col">操作</th>
      </tr></thead>
      <tbody>{skills.map((skill) => {
        const version = skill.defaultVersion;
        const tags = skill.tags ?? [];
        const purpose = skill.purpose || skill.description || "";
        return <tr key={skill.id} className="skillhub-entry" data-selected={selected.includes(skill.id) || undefined}>
          <td><input type="checkbox" aria-label={`选择 ${skill.name}`} checked={selected.includes(skill.id)}
            disabled={selected.length >= 100 && !selected.includes(skill.id)}
            onChange={() => setSelected((ids) => ids.includes(skill.id) ? ids.filter((id) => id !== skill.id) : [...ids, skill.id])} /></td>
          <td>
            <Link id={`skill-link-${skill.id}`} className="library-name" to={`/skills/${skill.id}`} state={{ library }}
              onClick={() => { libraryNavigation.returning = { scope, skills, token, scroll: window.scrollY, skillId: skill.id, revision: libraryNavigation.revision }; }}>{skill.name}</Link>
            {purpose && <span className="library-purpose" title={purpose}>{purpose}</span>}
            <span className="library-mobile-source">{[sourceLabel(skill), skill.category, ...tags.slice(0, 2)].filter(Boolean).join(" · ")}{tags.length > 2 && ` +${tags.length - 2}`}</span>
          </td>
          <td className="library-source">{sourceLabel(skill)}</td>
          <td className="library-classification">{[skill.category || "未分类", ...tags.slice(0, 2)].join(" · ")}{tags.length > 2 && ` +${tags.length - 2}`}</td>
          <td className="library-row-actions"><div className="library-row-controls">
            {version && (version.available
              ? <button type="button" className="skillhub-text-button" disabled={!!operation.busy}
                onClick={() => void operation.save(skill.id, version.id)}>{operation.busy === version.id ? "保存中…" : "保存到本机"}</button>
              : <button type="button" className="skillhub-text-button" disabled={!!operation.busy} onClick={() => void operation.run(version.id, async () => {
                await repairVersion({ skillId: skill.id, versionId: version.id });
                operation.setNotice("版本已补齐，可以保存到本机。");
                reload();
              })}>补齐版本</button>)}
          </div></td>
        </tr>;
      })}</tbody>
    </table>}
    {error?.pageToken && <Failure error={error.message} />}
    {token && <button className="secondary skillhub-more" type="button" disabled={loading} onClick={() => void load(token)}>
      {loading ? "加载中…" : error?.pageToken ? "重试加载更多" : "加载更多"}
    </button>}

    {panel?.kind === "folder" && <Dialog title="导入文件夹" compact onClose={() => setPanel(null)}>
      <LocalIntake onComplete={imported} onCancel={() => setPanel(null)}
        onVersion={(skill, path) => setPanel({ kind: "new-version", skill, path })} />
    </Dialog>}
    {panel?.kind === "github" && <Dialog title="导入 GitHub" compact onClose={() => setPanel(null)}><GitHubIntake onComplete={imported} /></Dialog>}
    {panel?.kind === "zip" && <Dialog title="导入 ZIP" compact onClose={() => setPanel(null)}><ZipIntake onComplete={imported} /></Dialog>}
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
        : <p>启动本机 Ollama，并在启动 skillhub 时设置 <code>SKILLHUB_OLLAMA_MODEL</code>。</p>}
    </Dialog>}
    {panel?.kind === "compare" && <Dialog title="比对 Skill" onClose={() => setPanel(null)}>
      <Suspense fallback={<p>加载中…</p>}><ComparePanel skills={panel.skills} /></Suspense>
    </Dialog>}
  </main>;
}
