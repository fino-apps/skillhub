import { useId, useState, type FormEvent, type ReactNode } from "react";
import { commitUpload, importSkill, listSkills, scanProject } from "@/generated/api-client";
import { backendDirectoryLabel, canPickDirectory, localCandidate, message, pickDirectory, stageZip, tagsFrom, type Candidate, type ImportResult, type LibraryPage, type Skill } from "./library-api";
import { Failure } from "./skill-ui";

const importLimit = 100;

export function CandidatePicker({ items, selected, onChange, disabled, single }: {
  items: Candidate[]; selected: string[]; onChange: (selection: string[]) => void; disabled?: boolean; single?: boolean;
}) {
  const group = useId();
  return <fieldset className="candidate-picker">
    <legend className="sr-only">选择要收录的技能</legend>
    {!single && <div className="library-actions candidate-selection">
      <span role="status">已选 {selected.length} / {items.length} 项{items.length > importLimit && "（每次最多 100 项）"}</span>
      <button type="button" className="skillhub-text-button" disabled={disabled || selected.length === Math.min(items.length, importLimit)}
        onClick={() => onChange(items.slice(0, importLimit).map(({ path }) => path ?? ""))}>
        {items.length > importLimit ? "选择前 100 项" : "全选"}
      </button>
      <button type="button" className="skillhub-text-button" disabled={disabled || !selected.length} onClick={() => onChange([])}>清空</button>
    </div>}
    <ul className="github-candidates">
      {items.map((item) => {
        const path = item.path ?? "";
        const checked = selected.includes(path);
        return <li key={path}><label className="github-checkbox">
          <input type={single ? "radio" : "checkbox"} name={group} checked={checked}
            disabled={disabled || (!checked && selected.length >= importLimit)}
            onChange={() => onChange(single ? [path] : checked ? selected.filter((value) => value !== path) : [...selected, path])} />
          <span><strong>{item.name}</strong><span className="candidate-description">{item.description}</span></span>
        </label></li>;
      })}
    </ul>
  </fieldset>;
}

export function ImportFeedback({ result }: { result: ImportResult | null }) {
  if (!result) return null;
  return <div className="skill-import-feedback">
    <p role="status">新增 {result.addedCount ?? 0} 项，已有 {result.existingCount ?? 0} 项{result.failures?.length ? `，失败 ${result.failures.length} 项` : ""}。</p>
    {result.failures && result.failures.length > 0 && <ul>{result.failures.map((failure) =>
      <li key={failure.path}>{failure.path} · {failure.message}</li>,
    )}</ul>}
  </div>;
}

export function ImportMetadataFields({ categories, disabled }: { categories: string[]; disabled: boolean }) {
  const hintId = useId();
  const categoriesId = useId();

  const [summary, setSummary] = useState("");

  return <details className="skill-disclosure file-import-metadata" onChange={(event) => {
    const form = event.currentTarget.closest("form");
    if (!form) return;
    const { category, tags } = importMetadataValues(form);
    setSummary([category, tags.length ? `${tags.length} 个标签` : ""].filter(Boolean).join(" · "));
  }}>
    <summary>分类和标签{summary ? <span title={summary}>{summary}</span> : "（选填）"}</summary>
    <fieldset className="skill-fields" disabled={disabled} aria-describedby={hintId}>
      <legend className="sr-only">批量分类和标签</legend>
      <span className="skillhub-hint" id={hintId}>应用到本批新收录的 Skill</span>
      <label>分类<input name="category" maxLength={128} list={categoriesId} /></label>
      <datalist id={categoriesId}>{categories.map((name) => <option key={name} value={name} />)}</datalist>
      <label>标签<input name="tags" placeholder="用逗号分隔" /></label>
    </fieldset>
  </details>;
}

function importMetadataValues(form: HTMLFormElement) {
  const values = new FormData(form);
  return {
    category: String(values.get("category") ?? "").trim(),
    tags: tagsFrom(String(values.get("tags") ?? "")),
  };
}

export function readImportMetadata(form: HTMLFormElement) {
  const { category, tags } = importMetadataValues(form);
  if (tags.length > 100 || tags.some((tag) => [...tag].length > 128)) {
    const details = form.querySelector<HTMLDetailsElement>(".file-import-metadata");
    if (details) details.open = true;
    throw new Error("最多填写 100 个标签，每个标签最多 128 个字符");
  }
  return { category, tags };
}

export default function LocalIntake({ categories, onComplete, onVersion, children }: {
  categories: string[];
  children?: ReactNode;
  onComplete: (result: ImportResult) => void; onVersion: (skill: Skill, path: string) => void;
}) {
  const [directory, setDirectory] = useState("");
  const [items, setItems] = useState<Candidate[]>([]);
  const [selected, setSelected] = useState<string[]>([]);
  const [matching, setMatching] = useState<Skill[]>([]);
  const [result, setResult] = useState<ImportResult | null>(null);
  const [busy, setBusy] = useState(false);
  const [error, setError] = useState("");
  const [manual, setManual] = useState(false);
  const pathId = useId();

  async function choose() {
    await run(async () => {
      const path = await pickDirectory();
      if (!path) return;
      setDirectory(path);
      await inspect(path);
    });
  }

  function resetPreview() {
    setItems([]);
    setSelected([]);
    setMatching([]);
    setResult(null);
  }

  async function inspect(path: string) {
    resetPreview();
    const reply = await scanProject<{ skills?: Candidate[] }>({ projectPath: path });
    const found = (reply.skills ?? []).map((item) => ({ ...item, path: localCandidate(path, item.path) }));
    setItems(found);
    if (!found.length) {
      setError("此目录没有找到 Skill");
      return;
    }
    setSelected(found.slice(0, importLimit).map((item) => item.path));
    if (found.length === 1) {
      const listed = await listSkills<LibraryPage>({ keyword: found[0].name, pageSize: 100 });
      setMatching((listed.skills ?? []).filter((item) => item.name === found[0].name));
    }
  }

  async function run(action: () => Promise<void>) {
    setBusy(true);
    setError("");
    try {
      await action();
    } catch (err) {
      setError(message(err));
    } finally {
      setBusy(false);
    }
  }

  function submit(event: FormEvent<HTMLFormElement>) {
    event.preventDefault();
    void run(() => inspect(directory.trim()));
  }

  function save(event: FormEvent<HTMLFormElement>) {
    event.preventDefault();
    if (!selected.length || busy) return;
    const form = event.currentTarget;
    void run(async () => {
      const metadata = readImportMetadata(form);
      const reply = await importSkill<ImportResult>({ skillPaths: selected, ...metadata });
      if (reply.failures?.length) {
        setResult(reply);
        setSelected(reply.failures.map(({ path }) => path ?? ""));
      }
      onComplete(reply);
    });
  }

  const directoryForm = <form id={pathId} className="github-form" onSubmit={submit}>
    <label>{backendDirectoryLabel}<input value={directory} onChange={(event) => setDirectory(event.target.value)}
      autoFocus={canPickDirectory && manual} required /></label>
    <button type="submit" className={canPickDirectory ? "secondary" : undefined}>扫描目录</button>
  </form>;

  return <div className="skill-fields local-intake">
    {busy && !items.length && <p role="status">正在读取文件夹…</p>}
    <Failure error={error} />
    <ImportFeedback result={result} />
    {!busy && !items.length && <>
      {canPickDirectory && <button type="button" className="file-import-pick" onClick={() => void choose()}>选择文件夹</button>}
      {!canPickDirectory && directoryForm}
      <div className="file-import-options">
        {canPickDirectory && <button type="button" className="skillhub-text-button" aria-expanded={manual}
          aria-controls={manual ? pathId : undefined} onClick={() => setManual(!manual)}>手动输入路径</button>}
        {children}
      </div>
      {canPickDirectory && manual && directoryForm}
    </>}
    {items.length > 0 && <div className="file-import-source">
      <span title={directory}>{directory.trim().split(/[/\\]/).filter(Boolean).pop() || directory}</span>
      <button type="button" className="skillhub-text-button" disabled={busy} onClick={() => {
        setError("");
        resetPreview();
      }}>更换目录</button>
    </div>}
    {items.length === 1 && <p className="local-intake-candidate">
      <strong>{items[0].name}</strong>
      {items[0].description && <span className="candidate-description">{items[0].description}</span>}
    </p>}
    {matching.length > 0 && <div className="skill-fields">
      <p className="skillhub-hint">可保存为同名 Skill 的新版本，或在下方作为独立条目收录。</p>
      {matching.map((skill) => <div className="skill-fields" key={skill.id}>
        <button type="button" className="secondary" disabled={busy}
          onClick={() => onVersion(skill, selected[0])}>为 {skill.name} 保存新版本</button>
        <small>已有条目来源 {skill.upstreamUrl || skill.localPath || "当前库"}</small>
      </div>)}
    </div>}
    {items.length > 0 && <form className="github-form" onSubmit={save}>
      {items.length > 1 && <CandidatePicker items={items} selected={selected} onChange={setSelected} disabled={busy} />}
      <ImportMetadataFields categories={categories} disabled={busy} />
      <button type="submit" disabled={!selected.length || busy}>
        {busy ? "处理中…" : matching.length ? "作为独立条目收录" : items.length === 1 ? "确认收录" : `收录 ${selected.length} 项`}
      </button>
    </form>}
  </div>;
}

export function ZipIntake({ skillId, onComplete, onBack }: {
  skillId?: string; onComplete: (result: ImportResult, pending?: boolean) => void; onBack?: () => void;
}) {
  const [uploadId, setUploadId] = useState("");
  const [items, setItems] = useState<Candidate[]>([]);
  const [selected, setSelected] = useState<string[]>([]);
  const [busy, setBusy] = useState(false);
  const [error, setError] = useState("");
  const [result, setResult] = useState<ImportResult | null>(null);

  async function read(file?: File) {
    if (!file) return;
    setBusy(true);
    setError("");
    setResult(null);
    setItems([]);
    setUploadId("");
    try {
      const reply = await stageZip(file);
      const candidates = reply.skills ?? [];
      setUploadId(reply.uploadId);
      setItems(candidates);
      setSelected(candidates.slice(0, skillId ? 1 : importLimit).map((item) => item.path ?? ""));
    } catch (err) {
      setError(message(err));
    } finally {
      setBusy(false);
    }
  }

  async function commit() {
    setBusy(true);
    setError("");
    try {
      const reply = await commitUpload<ImportResult>({ uploadId, skillPaths: selected, targetSkillId: skillId });
      const failed = new Set(reply.failures?.map(({ path }) => path ?? "") ?? []);
      const remaining = items.filter((item) => !selected.includes(item.path ?? "") || failed.has(item.path ?? ""));
      setItems(remaining);
      setSelected(remaining.slice(0, skillId ? 1 : importLimit).map((item) => item.path ?? ""));
      const total = {
        ...reply,
        addedCount: (result?.addedCount ?? 0) + (reply.addedCount ?? 0),
        existingCount: (result?.existingCount ?? 0) + (reply.existingCount ?? 0),
      };
      setResult(total);
      onComplete(total, remaining.length > 0);
    } catch (err) {
      setError(message(err));
    } finally {
      setBusy(false);
    }
  }

  return <div className="skill-fields">
    <label>ZIP 文件<input type="file" accept=".zip,application/zip" autoFocus={!!onBack} disabled={busy} onChange={(event) => void read(event.target.files?.[0])} /></label>
    {items.length > 0 && <CandidatePicker items={items} selected={selected} disabled={busy} single={!!skillId} onChange={setSelected} />}
    <Failure error={error} />
    <ImportFeedback result={result} />
    {busy && <p role="status">处理中…</p>}
    {uploadId && <button type="button" disabled={busy || !selected.length} onClick={() => void commit()}>确认收录</button>}
    {onBack && <button type="button" className="skillhub-text-button" disabled={busy} onClick={onBack}>返回文件夹导入</button>}
  </div>;
}
