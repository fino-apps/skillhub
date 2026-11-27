import { useEffect, useRef, useState, type ReactNode } from "react";
import { Link, useSearchParams } from "react-router-dom";
import { batchDeleteSkills, listFacets, listSkills, repairVersion } from "@/generated/api-client";
import GitHubIntake from "./github-intake";
import LibraryBackup, { downloadLibraryBackup } from "./library-backup";
import { UploadIntake } from "./local-intake";
import { BatchMetadata } from "./skill-metadata";
import { importSourceName, libraryNavigation, message, sourceURL, type Facets, type ImportResult, type LibraryPage, type Skill } from "./library-api";
import { mergeSummaryText } from "./skill-merge";
import { Dialog, ExportVersionDialog, Failure, Menu, OperationFeedback, Popover, useSkillOperation } from "./skill-ui";

import "./skillhub.css";

type Panel =
  | { kind: "github" }
  | { kind: "backup" }
  | { kind: "export"; skillId: string; versionId: string }
  | { kind: "batch"; skills: Skill[] }
  | { kind: "file"; zip?: boolean };

function selectedSource(params: URLSearchParams) {
  const legacy = params.get("source");
  return params.get("sourceName") || (legacy ? importSourceName(legacy) : "");
}

function LibraryToolbar({ params, facets, onChange, status, children }: {
  params: URLSearchParams; facets: Facets; onChange: (next: URLSearchParams) => void; status: ReactNode; children: ReactNode;
}) {
  const [composition, setComposition] = useState<string | null>(null);
  const composing = useRef(false);
  const query = params.get("q") ?? "";
  const source = selectedSource(params);
  const tags = params.getAll("tag");
  const category = params.get("category") ?? "";
  const tagOptions = Array.from(new Set([...(facets.tags ?? []), ...tags]));

  function set(name: string, value: string) {
    const next = new URLSearchParams(params);
    if (name === "sourceName") next.delete("source");
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
    <div className="library-toolbar">
      <h1 id="skill-library-heading" className="library-heading" tabIndex={-1}>技能库</h1>
      <label className="skillhub-search">
        <span aria-hidden="true">⌕</span><span className="sr-only">搜索 Skill</span>
        <input type="search" name="q" autoComplete="off" autoCorrect="off" autoCapitalize="none" spellCheck={false}
          placeholder="搜索名称、用途或标签…" value={composition ?? query}
          onCompositionStart={(event) => {
            composing.current = true;
            setComposition(event.currentTarget.value);
          }}
          onCompositionEnd={(event) => {
            composing.current = false;
            setComposition(null);
            if (event.currentTarget.value !== query) set("q", event.currentTarget.value);
          }}
          onChange={(event) => {
            const value = event.currentTarget.value;
            if (composing.current) setComposition(value);
            else if (value !== query) set("q", value);
          }} />
      </label>
      <div className="library-facets" role="group" aria-label="筛选技能">
        <label className="library-facet" data-active={!!source || undefined}>
          <span className="library-facet-text" aria-hidden="true">{source || "全部来源"}</span>
          <select aria-label="来源" value={source} onChange={(event) => set("sourceName", event.target.value)}>
            <option value="">全部来源</option>
            {source && !facets.sourceNames?.includes(source) && <option value={source}>{source}</option>}
            {facets.sourceNames?.map((name) => <option key={name} value={name}>{name}</option>)}
          </select>
        </label>
        <label className="library-facet" data-active={!!category || undefined}>
          <span className="library-facet-text" aria-hidden="true">{category === "~" ? "未分类" : category || "全部分类"}</span>
          <select aria-label="分类" value={category} onChange={(event) => set("category", event.target.value)}>
            <option value="">全部分类</option><option value="~">未分类</option>
            {category && category !== "~" && !facets.categories?.includes(category) && <option>{category}</option>}
            {facets.categories?.map((category) => <option key={category}>{category}</option>)}
          </select>
        </label>
        <Popover className={`skill-filter${tags.length ? " has-filters" : ""}`} summary={<span className="library-facet-text">标签</span>}>
          <div className="filter-panel">
            <fieldset className="skill-fields filter-tags">
              <legend className="sr-only">标签</legend>
              {tagOptions.length === 0 && <p>暂无可筛选的标签</p>}
              {tagOptions.map((tag) => <label key={tag} className="github-checkbox">
                <input type="checkbox" checked={tags.includes(tag)} onChange={(event) =>
                  setTags(event.target.checked ? [...tags, tag] : tags.filter((value) => value !== tag))} />{tag}
              </label>)}
            </fieldset>
          </div>
        </Popover>
      </div>
      <div className="library-filter-status">
        {status}
        {params.size > 0 && <button type="button" className="skillhub-text-button"
          onClick={() => onChange(new URLSearchParams())}>清除筛选</button>}
      </div>
      {children}
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
    void listFacets<Facets>({}, { signal: controller.signal }).then((result) => {
      if (!controller.signal.aborted) setFacets({ ...result, sourceNames: result.sourceNames ?? [] });
    }, () => {});
    return () => controller.abort();
  }, [refresh]);

  useEffect(() => {
    const source = selectedSource(params);
    if (!source || !facets.sourceNames || facets.sourceNames.includes(source)) return;
    const next = new URLSearchParams(params);
    next.delete("sourceName");
    next.delete("source");
    setParams(next, { replace: true });
  }, [facets.sourceNames, params, setParams]);

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
      sourceName: selectedSource(params), tags: params.getAll("tag"), uncategorized: category === "~", pageSize: 40,
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

  function imported(result: ImportResult, pending = false) {
    reload();
    if (pending || result.failures?.length) return;
    setPanel(null);
    operation.setNotice(`新增 ${result.addedCount ?? 0} 项，已有 ${result.existingCount ?? 0} 项。`);
  }

  function exportBackup() {
    void operation.run("backup-export", async () => {
      await downloadLibraryBackup();
      operation.setNotice("已发起备份下载。");
    });
  }

  function removeSelected() {
    if (!window.confirm(`删除选中的 ${selected.length} 项 Skill 及其已保存版本？`)) return;
    void operation.run("batch-delete", async () => {
      await batchDeleteSkills({ skillIds: selected });
      libraryNavigation.revision += 1;
      reload();
      operation.setNotice(`已删除 ${selected.length} 项 Skill。`);
    });
  }

  const selectable = skills.slice(0, 100);
  const allSelected = selectable.length > 0 && selectable.every(({ id }) => selected.includes(id));
  useEffect(() => {
    if (selectAll.current) selectAll.current.indeterminate = selected.length > 0 && !allSelected;
  }, [selected.length, allSelected, current]);
  const library = scope ? `/?${scope}` : "/";
  const editComplete = () => { setPanel(null); reload(); };

  return <main className="skillhub-shell library-shell">
    <LibraryToolbar params={params} facets={facets} onChange={updateFilters} status={
      <span role="status">{!current && loading
        ? "正在查找技能…" : current && `${token ? "已加载" : "共"} ${skills.length} 项`}</span>
    }>
      <div className="library-actions library-import-actions">
        <button type="button" onClick={() => setPanel({ kind: "file" })}>导入文件</button>
        <button type="button" className="secondary" onClick={() => setPanel({ kind: "github" })}>导入 GitHub</button>
        <Menu label="备份与恢复">
          <button type="button" disabled={!!operation.busy} onClick={exportBackup}>
            {operation.busy === "backup-export" ? "正在导出备份…" : "导出技能库备份"}
          </button>
          <button type="button" onClick={() => setPanel({ kind: "backup" })}>导入备份</button>
        </Menu>
      </div>
    </LibraryToolbar>
    <div className="library-overlays">
      {panel?.kind !== "export" && <OperationFeedback {...operation} floating />}
      {selected.length > 0 && <div className="library-actions library-selection" role="group" aria-label="批量操作">
        <span role="status">已选 {selected.length} 项</span>
        <button type="button" className="secondary" onClick={() => setPanel({ kind: "batch", skills: skills.filter(({ id }) => selected.includes(id)) })}>批量编辑</button>
        <button type="button" className="secondary" disabled={!!operation.busy} onClick={removeSelected}>批量删除</button>
        <button type="button" className="skillhub-text-button" onClick={() => {
          setSelected([]);
          selectAll.current?.focus({ preventScroll: true });
        }}>清除选择</button>
      </div>}
    </div>
    {error && !error.pageToken && <div className="library-actions"><Failure error={error.message} /><button type="button" className="secondary" onClick={() => void load("")}>重试</button></div>}
    {!loading && !error && skills.length === 0 && <div className="skillhub-empty">
      <p>{scope ? "没有匹配的 Skill" : "还没有 Skill"}</p>
    </div>}
    {skills.length > 0 && <table className="skillhub-table library-table" aria-busy={loading}>
      <thead><tr>
        <th scope="col"><input ref={selectAll} type="checkbox" aria-label={skills.length > 100 ? "选择前 100 项" : "选择当前已加载条目"} checked={allSelected}
          onChange={() => setSelected(allSelected ? [] : selectable.map(({ id }) => id))} /></th>
        <th scope="col">Skill / 用途</th>
        <th scope="col" className="library-source">来源</th>
        <th scope="col" className="library-classification">分类 / 标签</th>
        <th scope="col">操作</th>
      </tr></thead>
      <tbody>{skills.map((skill) => {
        const version = skill.defaultVersion;
        const source = skill.upstreamName?.trim() || importSourceName(skill.importKind);
        const url = sourceURL(skill);
        const destination = url.replace(/^https?:\/\/(?:github\.com\/(?=.+))?/i, "");
        const sourceLink = url && (
          <a className="library-source-link" href={url} title={url} target="_blank" rel="noreferrer">{destination}</a>
        );
        const classification = [skill.category, ...(skill.tags ?? [])].filter(Boolean).join(" · ");
        const description = skill.description;
        return <tr key={skill.id} className="skillhub-entry" data-selected={selected.includes(skill.id) || undefined}>
          <td><input type="checkbox" aria-label={`选择 ${skill.name}`} checked={selected.includes(skill.id)}
            disabled={selected.length >= 100 && !selected.includes(skill.id)}
            onChange={() => setSelected((ids) => ids.includes(skill.id) ? ids.filter((id) => id !== skill.id) : [...ids, skill.id])} /></td>
          <td>
            <div className="skill-name-line">
              <Link id={`skill-link-${skill.id}`} className="library-name" to={`/skills/${skill.id}`} state={{ library }}
                onClick={() => { libraryNavigation.returning = { scope, skills, token, scroll: window.scrollY, skillId: skill.id, revision: libraryNavigation.revision }; }}>{skill.name}</Link>
              {skill.purpose && <span className="skill-short-purpose">{skill.purpose}</span>}
            </div>
            {description && <span className="library-purpose" title={description}>{description}</span>}
            <span className="library-mobile-source" title={classification ? `${source} · ${classification}` : undefined}>
              {source}{sourceLink && <> · {sourceLink}</>}{classification && ` · ${classification}`}
            </span>
          </td>
          <td className="library-source">
            <span className="library-source-kind">{source}</span>
            {sourceLink}
          </td>
          <td className="library-classification"><span title={classification || undefined}>{classification || "未分类"}</span></td>
          <td className="library-row-actions"><div className="library-row-controls">
            {version && (version.available
              ? <button type="button" className="skillhub-text-button" disabled={!!operation.busy}
                onClick={() => { operation.dismiss(); setPanel({ kind: "export", skillId: skill.id, versionId: version.id }); }}>导出</button>
              : <button type="button" className="skillhub-text-button" disabled={!!operation.busy} onClick={() => void operation.run(version.id, async () => {
                await repairVersion({ skillId: skill.id, versionId: version.id });
                operation.setNotice("版本已补齐，可以导出。");
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

    {panel?.kind === "file" && <Dialog title="导入文件" compact onClose={() => setPanel(null)}>
      <UploadIntake key={panel.zip ? "zip" : "directory"} directory={!panel.zip}
        categories={facets.categories ?? []} existingTags={facets.tags ?? []} onComplete={imported}>
        <button type="button" className="skillhub-text-button" onClick={() => setPanel({ kind: "file", zip: !panel.zip })}>
          {panel.zip ? "改用本机目录" : "改用 ZIP 文件"}
        </button>
      </UploadIntake>
    </Dialog>}
    {panel?.kind === "export" && <ExportVersionDialog skillId={panel.skillId} versionId={panel.versionId} operation={operation} onClose={() => setPanel(null)} />}
    {panel?.kind === "github" && <Dialog title="导入 GitHub" compact onClose={() => setPanel(null)}>
      <GitHubIntake categories={facets.categories ?? []} existingTags={facets.tags ?? []} onComplete={imported} />
    </Dialog>}
    {panel?.kind === "backup" && <Dialog title="导入备份" compact onClose={() => setPanel(null)}>
      <LibraryBackup onImported={(result) => {
        setPanel((current) => current?.kind === "backup" ? null : current);
        libraryNavigation.revision += 1;
        reload();
        const stats = `新增 ${result.addedCount ?? 0} 项，合并 ${result.mergedCount ?? 0} 项，跳过 ${result.skippedCount ?? 0} 项。`;
        operation.setNotice((result.mergedCount ?? 0) > 0 && result.versions ? `${stats}${mergeSummaryText(result.versions)}` : stats);
      }} />
    </Dialog>}
    {panel?.kind === "batch" && <Dialog title="批量编辑" compact onClose={() => setPanel(null)}>
      <BatchMetadata skills={panel.skills} categories={facets.categories ?? []} onSaved={() => {
        editComplete();
        operation.setNotice(`已更新 ${panel.skills.length} 项 Skill。`);
      }} />
    </Dialog>}
  </main>;
}
