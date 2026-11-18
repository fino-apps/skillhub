import { lazy, Suspense, useEffect, useState } from "react";
import { Link, useLocation, useNavigate, useParams, useSearchParams } from "react-router-dom";
import { deleteSkill, deleteVersion, getSkill, listFacets, repairVersion, setDefaultVersion } from "@/generated/api-client";
import InstallCommand from "./install-command";
import { ZipIntake } from "./local-intake";
import SkillContent from "./skill-content";
import SkillMetadata from "./skill-metadata";
import { archiveURL, fetchAllVersions, libraryNavigation, message, sourceLabel, sourceNames, sourceURL, type Facets, type Skill } from "./library-api";
import { Dialog, ExportLocation, Failure, Menu, OperationFeedback, useSkillOperation } from "./skill-ui";
import { shortVersion, versionDigest, versionLabel, type SkillVersion } from "./skill-version";

import "./skillhub.css";

const ComparePanel = lazy(() => import("./compare-panel"));
const VersionPanel = lazy(() => import("./version-panel"));

type Panel =
  | { kind: "metadata" | "compare" | "zip" }
  | { kind: "export" | "install"; version: SkillVersion }
  | { kind: "update"; source: "github" | "folder" };

export default function SkillDetail() {
  const { skillId = "" } = useParams();
  return <Detail key={skillId} skillId={skillId} />;
}

function Detail({ skillId }: { skillId: string }) {
  const [params, setParams] = useSearchParams();
  const navigate = useNavigate();
  const location = useLocation();
  const back = typeof location.state?.library === "string" && location.state.library.startsWith("/?") ? location.state.library : "/";
  const [skill, setSkill] = useState<Skill | null>(null);
  const [versions, setVersions] = useState<SkillVersion[]>([]);
  const [categories, setCategories] = useState<string[]>([]);
  const [refresh, setRefresh] = useState(0);
  const [error, setError] = useState("");
  const [panel, setPanel] = useState<Panel | null>(null);
  const operation = useSkillOperation();

  useEffect(() => {
    const controller = new AbortController();
    setError("");
    void Promise.all([getSkill<{ skill: Skill }>({ skillId }, { signal: controller.signal }), fetchAllVersions(skillId, controller.signal)])
      .then(([result, versions]) => {
        if (controller.signal.aborted) return;
        setSkill(result.skill);
        setVersions(versions);
      }, (err) => { if (!controller.signal.aborted) setError(message(err)); });
    return () => controller.abort();
  }, [skillId, refresh]);

  useEffect(() => {
    const controller = new AbortController();
    void listFacets<Facets>({}, { signal: controller.signal }).then((result) => setCategories(result.categories ?? []), () => {});
    return () => controller.abort();
  }, [refresh]);

  function reload() {
    libraryNavigation.revision += 1;
    setRefresh((value) => value + 1);
  }

  function saved() {
    setPanel(null);
    reload();
  }

  function routeValue(name: string, value: string) {
    const next = new URLSearchParams(params);
    if (value) next.set(name, value);
    else next.delete(name);
    setParams(next, { replace: true, state: location.state });
  }

  const view = versions.find((item) => item.id === params.get("version")) ?? skill?.defaultVersion;
  const versionsTab = params.get("tab") === "versions";
  const busy = !!operation.busy;
  const updateSource = skill?.repositoryUrl ? "github" : "folder";
  const canCompare = versions.filter((version) => version.available).length >= 2;
  const description = skill?.description || "";

  function repair(version: SkillVersion) {
    void operation.run(version.id, async () => {
      await repairVersion({ skillId, versionId: version.id });
      operation.setNotice("版本已补齐。");
      reload();
    });
  }

  function makeDefault(version: SkillVersion) {
    void operation.run(version.id, async () => {
      await setDefaultVersion({ skillId, versionId: version.id });
      operation.setNotice("已设为默认版本。");
      reload();
    });
  }

  function removeVersion(version: SkillVersion) {
    if (!window.confirm(`删除版本 ${shortVersion(version, versions)}？`)) return;
    void operation.run(version.id, async () => {
      await deleteVersion({ skillId, versionId: version.id });
      if (view?.id === version.id) routeValue("version", "");
      reload();
    });
  }

  function removeSkill(name: string) {
    if (!window.confirm(`删除「${name}」及其已保存版本？`)) return;
    void operation.run(skillId, async () => {
      await deleteSkill({ skillId });
      libraryNavigation.revision += 1;
      navigate(back);
    });
  }

  const url = skill && sourceURL(skill);
  return <main className="skillhub-shell skill-detail-shell">
    <Link className="skill-back" to={back}>‹ 技能库</Link>
    <Failure error={error} />
    {error && <button type="button" className="secondary" onClick={reload}>重试</button>}
    {!skill && !error && <p role="status">读取中…</p>}
    {skill && <>
      <div className="skillhub-heading skill-detail-heading">
        <div className="skill-identity skill-name-line">
          <h1>{skill.name}</h1>
          {skill.purpose && <span className="skill-short-purpose">{skill.purpose}</span>}
        </div>
        <div className="library-actions">
          {view && (view.available
            ? <button type="button" disabled={busy} onClick={() => void operation.save(skillId, view.id)}>{busy ? "处理中…" : "保存到本机"}</button>
            : <button type="button" disabled={busy} onClick={() => repair(view)}>补齐版本</button>)}
          <Menu label="Skill 更多操作">
            <button type="button" onClick={() => setPanel({ kind: "metadata" })}>编辑整理信息</button>
            {view?.available && <>
              <button type="button" onClick={() => { operation.dismiss(); setPanel({ kind: "export", version: view }); }}>另存到…</button>
              <a href={archiveURL(skillId, view.id)}>下载 ZIP</a>
              <button type="button" onClick={() => setPanel({ kind: "install", version: view })}>复制安装命令</button>
            </>}
            <button type="button" className="skill-danger" disabled={busy} onClick={() => removeSkill(skill.name)}>删除 Skill</button>
          </Menu>
        </div>
      </div>
      <div className="skill-detail-meta">
        {view && <span>当前版本 <code title={versionDigest(view)}>{shortVersion(view, versions)}</code></span>}
        <details className="skill-disclosure">
          <summary>来源 {sourceLabel(skill)}</summary>
          <dl>
            <dt>导入方式</dt><dd>{sourceNames[skill.importKind ?? ""] || "未知"}</dd>
            {url && <><dt>来源</dt><dd><a href={url} target="_blank" rel="noreferrer">{url}</a></dd></>}
            {skill.repositoryUrl && <><dt>绑定仓库</dt><dd>{skill.repositoryUrl}{skill.repositoryPath && <code> · {skill.repositoryPath}</code>}</dd></>}
            {skill.localPath && <><dt>绑定目录</dt><dd><code>{skill.localPath}</code></dd></>}
            {view?.upstreamRepository && <><dt>归档来源</dt><dd>{view.upstreamRepository}{view.upstreamPath && <code> · {view.upstreamPath}</code>}</dd></>}
            {view?.upstreamCommit && <><dt>归档来源 Commit</dt><dd><code>{view.upstreamCommit}</code></dd></>}
          </dl>
        </details>
        {skill.category && <span>{skill.category}</span>}
        {skill.tags?.map((tag) => <span key={tag}>{tag}</span>)}
      </div>
      {description && <p className="skill-detail-purpose">{description}</p>}
      {panel?.kind !== "export" && <OperationFeedback {...operation} floating />}
      <nav className="skill-tabs" aria-label="Skill 详情">
        <button type="button" aria-pressed={!versionsTab} onClick={() => routeValue("tab", "")}>内容</button>
        <button type="button" aria-pressed={versionsTab} onClick={() => routeValue("tab", "versions")}>版本</button>
      </nav>
      {!versionsTab && (view
        ? <SkillContent key={view.id} skillId={skillId} skillName={skill.name} versionId={view.id} available={!!view.available} description={description} />
        : <p className="skillhub-empty">暂无版本</p>)}
      {versionsTab && <section className="skill-versions">
        <div className="library-actions">
          <button type="button" className="secondary" onClick={() => setPanel({ kind: "update", source: updateSource })}>
            {updateSource === "github" ? "检查更新" : "保存新版本"}
          </button>
          {canCompare && <button type="button" className="secondary" onClick={() => setPanel({ kind: "compare" })}>比对版本</button>}
          <Menu label="版本更多操作">
            {updateSource === "github" && <button type="button" onClick={() => setPanel({ kind: "update", source: "folder" })}>从文件夹保存版本</button>}
            <button type="button" onClick={() => setPanel({ kind: "zip" })}>从 ZIP 保存版本</button>
          </Menu>
        </div>
        {versions.length === 0 && <p className="skillhub-empty">暂无版本</p>}
        <ul className="version-history">
          {versions.map((version) => {
            const isDefault = skill.defaultVersion?.id === version.id;
            return <li key={version.id}>
              <button type="button" className="skillhub-text-button" title={versionDigest(version)}
                aria-pressed={view?.id === version.id} onClick={() => routeValue("version", version.id)}><code>{shortVersion(version, versions)}</code></button>
              <span>{isDefault ? "默认" : ""}{view?.id === version.id && !isDefault ? "正在查看" : ""}</span>
              <time dateTime={version.createTime}>{versionLabel(version)}</time>
              <span className="version-notes-text">{version.available ? version.notes : "待补齐"}</span>
              {!isDefault && <Menu label={`版本 ${shortVersion(version, versions)} 管理`}>
                {version.available && <button type="button" disabled={busy} onClick={() => makeDefault(version)}>设为默认</button>}
                <button type="button" disabled={busy} className="skill-danger" onClick={() => removeVersion(version)}>删除版本</button>
              </Menu>}
            </li>;
          })}
        </ul>
      </section>}
      {panel?.kind === "metadata" && <Dialog title="编辑整理信息" compact onClose={() => setPanel(null)}>
        <SkillMetadata skill={skill} categories={categories} onSaved={saved} />
      </Dialog>}
      {panel?.kind === "install" && <Dialog title="Codex 安装命令" compact onClose={() => setPanel(null)}>
        <InstallCommand skillId={skillId} versionId={panel.version.id} />
      </Dialog>}
      {panel?.kind === "export" && <Dialog title="另存到本机" compact onClose={() => setPanel(null)}>
        <ExportLocation busy={busy} onSave={(dir) => void operation.save(skillId, panel.version.id, dir === undefined, dir)} />
        <OperationFeedback {...operation} />
      </Dialog>}
      {panel?.kind === "zip" && <Dialog title="从 ZIP 保存版本" compact onClose={() => setPanel(null)}>
        <ZipIntake skillId={skillId} onComplete={(result, pending) => { if (!pending && !result.failures?.length) saved(); else reload(); }} />
      </Dialog>}
      {panel?.kind === "update" && <Dialog title={panel.source === "github" ? "检查更新" : "保存新版本"} onClose={() => setPanel(null)}>
        <Suspense fallback={<p>加载中…</p>}><VersionPanel skill={skill} from={panel.source} onSaved={(version) => {
          routeValue("version", version.id);
          saved();
        }} /></Suspense>
      </Dialog>}
      {panel?.kind === "compare" && <Dialog title="比对版本" onClose={() => setPanel(null)}>
        <Suspense fallback={<p>加载中…</p>}><ComparePanel skills={[skill]} skillId={skillId} candidate={view?.id !== skill.defaultVersion?.id ? view?.id : ""} /></Suspense>
      </Dialog>}
    </>}
  </main>;
}
