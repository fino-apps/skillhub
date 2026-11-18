import { useEffect, useState, type FormEvent } from "react";
import ReactDiffViewer from "react-diff-viewer-continued";
import { compareSkills, scanProject } from "@/generated/api-client";
import { shortVersion, versionLabel, type SkillVersion } from "./skill-version";
import { backendDirectoryLabel, canPickDirectory, fetchAllSkills, fetchAllVersions, localCandidate, pickDirectory, type Skill } from "./library-api";
type Source = { skillId: string; versionId: string; path: string };
export type Change = { path: string; kind: string; binary?: boolean; leftText?: string; rightText?: string; detail?: string };

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
      setFeedback("无法扫描，请检查目录路径。");
    } finally {
      setScanning(false);
    }
  }

  return <div className="compare-local">
    <label>{backendDirectoryLabel}<input value={path} onChange={(event) => { onChange(event.target.value); setFound([]); setFeedback(""); }} placeholder="/path/to/project" required disabled={scanning} /></label>
    {canPickDirectory && <button type="button" className="secondary" disabled={scanning} onClick={() => {
      void pickDirectory().then((dir) => { if (dir) { onChange(dir); setFound([]); } }, () => setFeedback("无法打开目录选择器，请输入路径。"));
    }}>选择文件夹</button>}
    <button type="button" className="secondary" disabled={scanning || !path.trim()} onClick={() => void scan()}>{scanning ? "扫描中…" : "扫描项目"}</button>
    {found.length > 0 && <label>选择 Skill<select defaultValue="" onChange={(event) => {
      const relative = event.target.value;
      if (relative) onChange(localCandidate(base, relative));
    }}>
      <option value="" disabled>选择扫描结果</option>
      {found.map((skill) => <option key={skill.path} value={skill.path}>{skill.name}</option>)}
    </select></label>}
    {feedback && <p className="skillhub-hint" role="status">{feedback}</p>}
  </div>;
}

function VersionSelect({ skill, versions, value, onChange }: {
  skill: Skill; versions: SkillVersion[]; value: string; onChange: (value: string) => void;
}) {
  return <label>版本<select value={value} onChange={(event) => onChange(event.target.value)}>
    <option value="" disabled={!skill.defaultVersion?.available}>
      {skill.defaultVersion?.available ? "默认版本" : "默认版本不可用"}
    </option>
    {versions.filter((version) => version.available && version.id !== skill.defaultVersion?.id).map((version) =>
      <option key={version.id} value={version.id}>{versionLabel(version)} · {shortVersion(version, versions)}</option>,
    )}
  </select></label>;
}

function OtherSkillVersion({ skill, value, onChange }: {
  skill: Skill; value: string; onChange: (value: string) => void;
}) {
  const [versions, setVersions] = useState<SkillVersion[]>([]);
  const [error, setError] = useState("");

  useEffect(() => {
    const controller = new AbortController();
    void fetchAllVersions(skill.id, controller.signal).then(
      (items) => { if (!controller.signal.aborted) setVersions(items); },
      () => { if (!controller.signal.aborted) setError("版本加载失败，请重新选择 Skill。"); },
    );
    return () => controller.abort();
  }, [skill.id]);

  return <>
    <VersionSelect skill={skill} versions={versions} value={value} onChange={onChange} />
    {error && <p role="alert" className="intake-error">{error}</p>}
  </>;
}

function location(source: Source) {
  if (source.versionId) return { versionId: source.versionId };
  if (source.skillId) return { currentSkillId: source.skillId };
  return { localPath: source.path.trim() };
}

export default function ComparePanel({ skill, versions, candidate }: { skill: Skill; versions: SkillVersion[]; candidate?: string }) {
  const defaultId = skill.defaultVersion?.id;
  const [left, setLeft] = useState(() => skill.defaultVersion?.available ? "" : versions.find((version) => version.available)?.id ?? "");
  const [right, setRight] = useState<Source>({ skillId: skill.id, versionId: candidate && candidate !== defaultId ? candidate : left, path: "" });
  const [skills, setSkills] = useState<Skill[]>([]);
  const [loadingSkills, setLoadingSkills] = useState(true);
  const [skillsError, setSkillsError] = useState("");
  const [changes, setChanges] = useState<Change[] | null>(null);
  const [busy, setBusy] = useState(false);
  const [error, setError] = useState("");

  useEffect(() => {
    const controller = new AbortController();
    void fetchAllSkills(controller.signal).then(
      (items) => { if (!controller.signal.aborted) setSkills(items.filter((item) => item.id !== skill.id)); },
      () => { if (!controller.signal.aborted) setSkillsError("其他 Skill 加载失败，请重新打开比对。"); },
    ).finally(() => { if (!controller.signal.aborted) setLoadingSkills(false); });
    return () => controller.abort();
  }, [skill.id]);

  const target = right.skillId === skill.id ? skill : skills.find((item) => item.id === right.skillId);
  const sameVersion = target === skill && left === right.versionId;
  const canSubmit = right.skillId
    ? !!(right.versionId || target?.defaultVersion?.available) && !sameVersion
    : !!right.path.trim();

  function changeTarget(value: Source) {
    setRight(value);
    setChanges(null);
    setError("");
  }

  async function submit(event: FormEvent<HTMLFormElement>) {
    event.preventDefault();
    if (busy || !canSubmit) return;
    setBusy(true);
    setError("");
    setChanges(null);
    try {
      const result = await compareSkills<{ files?: Change[] }>({ left: location({ skillId: skill.id, versionId: left, path: "" }), right: location(right) });
      setChanges(result.files ?? []);
    } catch (reason) {
      setError(reason instanceof Error && reason.message !== "bad request" ? reason.message : "无法比对，请检查目录及文件大小。");
    } finally { setBusy(false); }
  }

  return <div className="compare-panel">
    <form onSubmit={submit}>
      <div className="compare-sources">
        <fieldset className="compare-source" disabled={busy}>
          <legend>基准 · {skill.name}</legend>
          <VersionSelect skill={skill} versions={versions} value={left} onChange={(value) => {
            setLeft(value);
            setChanges(null);
            setError("");
          }} />
        </fieldset>
        <fieldset className="compare-source" disabled={busy}>
          <legend>对比对象</legend>
          <label>来源<select value={right.skillId} onChange={(event) => {
            const id = event.target.value;
            changeTarget({ ...right, skillId: id, versionId: id === skill.id && !skill.defaultVersion?.available ? left : "" });
          }}>
            <option value={skill.id}>当前 Skill 的版本</option>
            <option value="">本地目录</option>
            {skills.length > 0 && <optgroup label="其他 Skill">
              {skills.map((item) => <option key={item.id} value={item.id}>{item.name}</option>)}
            </optgroup>}
          </select></label>
          {loadingSkills && <p className="skillhub-hint" role="status">正在加载其他 Skill…</p>}
          {skillsError && <p className="intake-error" role="alert">{skillsError}</p>}
          {target === skill && <VersionSelect skill={skill} versions={versions} value={right.versionId}
            onChange={(versionId) => changeTarget({ ...right, versionId })} />}
          {target && target !== skill && <OtherSkillVersion key={target.id} skill={target} value={right.versionId}
            onChange={(versionId) => changeTarget({ ...right, versionId })} />}
          {!right.skillId && <LocalSource path={right.path} onChange={(path) => changeTarget({ ...right, path })} />}
          {sameVersion && <p className="skillhub-hint">请选择不同的版本或切换来源。</p>}
        </fieldset>
      </div>
      <button type="submit" disabled={busy || !canSubmit}>{busy ? "比对中…" : "开始比对"}</button>
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
    <div className="compare-content">
      {selected.detail && <p>{selected.detail}</p>}
      {selected.binary ? <p>二进制文件已变化。</p> : (selected.leftText !== undefined || selected.rightText !== undefined || !selected.detail)
        && <ReactDiffViewer oldValue={selected.leftText ?? ""} newValue={selected.rightText ?? ""} splitView={false} useDarkTheme={dark} />}
    </div>
  </div>;
}
