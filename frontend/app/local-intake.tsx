import { useEffect, useId, useRef, useState, type FormEvent } from "react";
import { commitUpload, importSkill, listSkills, scanProject } from "@/generated/api-client";
import { localCandidate, message, pickDirectory, stageZip, type Candidate, type ImportResult, type LibraryPage, type Skill } from "./library-api";
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

export default function LocalIntake({ onComplete, onCancel, onVersion }: {
  onComplete: (result: ImportResult) => void; onCancel: () => void; onVersion: (skill: Skill, path: string) => void;
}) {
  const [directory, setDirectory] = useState("");
  const [items, setItems] = useState<Candidate[]>([]);
  const [selected, setSelected] = useState<string[]>([]);
  const [matching, setMatching] = useState<Skill[]>([]);
  const [result, setResult] = useState<ImportResult | null>(null);
  const [busy, setBusy] = useState(true);
  const [error, setError] = useState("");
  const invoked = useRef(false);

  useEffect(() => {
    if (invoked.current) return;
    invoked.current = true;
    void choose();
  }, []);

  async function choose() {
    setBusy(true);
    setError("");
    try {
      const path = await pickDirectory();
      if (!path) {
        onCancel();
        return;
      }
      setDirectory(path);
      await inspect(path);
    } catch (err) {
      setError(message(err));
    } finally {
      setBusy(false);
    }
  }

  async function inspect(path: string) {
    setItems([]);
    setSelected([]);
    setMatching([]);
    setResult(null);
    const reply = await scanProject<{ skills?: Candidate[] }>({ projectPath: path });
    const found = (reply.skills ?? []).map((item) => ({ ...item, path: localCandidate(path, item.path) }));
    setItems(found);
    if (!found.length) {
      setError("此目录没有找到 Skill");
      return;
    }
    const selection = found.slice(0, importLimit).map((item) => item.path);
    setSelected(selection);
    if (found.length === 1) {
      const listed = await listSkills<LibraryPage>({ keyword: found[0].name, pageSize: 100 });
      const matches = (listed.skills ?? []).filter((item) => item.name === found[0].name);
      if (matches.length) {
        setMatching(matches);
      } else {
        await commit(selection);
      }
    }
  }

  async function commit(paths: string[]) {
    const reply = await importSkill<ImportResult>({ skillPaths: paths });
    if (reply.failures?.length) {
      setResult(reply);
      setSelected(reply.failures.map(({ path }) => path ?? ""));
    }
    onComplete(reply);
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

  return <>
    {busy && <p role="status">正在读取文件夹…</p>}
    <Failure error={error} />
    <ImportFeedback result={result} />
    {!busy && !items.length && <form className="github-form" onSubmit={submit}>
      <label>本机目录<input value={directory} onChange={(event) => setDirectory(event.target.value)} required /></label>
      <div className="library-actions">
        <button type="submit">扫描目录</button>
        <button type="button" className="secondary" onClick={() => void choose()}>选择文件夹</button>
      </div>
    </form>}
    {matching.length > 0 && <div className="skill-fields">
      <p>已有同名 Skill</p>
      {matching.map((skill) => <button key={skill.id} type="button" className="secondary" disabled={busy}
        onClick={() => onVersion(skill, selected[0])}>保存到 {skill.name} · {skill.upstreamName || skill.localPath || "当前库"}</button>)}
      <button type="button" disabled={busy} onClick={() => void run(() => commit(selected))}>作为独立条目收录</button>
    </div>}
    {!matching.length && items.length > 0 && <div className="skill-fields">
      <CandidatePicker items={items} selected={selected} onChange={setSelected} disabled={busy} />
      <button type="button" disabled={!selected.length || busy} onClick={() => void run(() => commit(selected))}>收录所选（{selected.length}）</button>
    </div>}
  </>;
}

export function ZipIntake({ skillId, onComplete }: { skillId?: string; onComplete: (result: ImportResult, pending?: boolean) => void }) {
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
    <label>ZIP 文件<input type="file" accept=".zip,application/zip" disabled={busy} onChange={(event) => void read(event.target.files?.[0])} /></label>
    {items.length > 0 && <CandidatePicker items={items} selected={selected} disabled={busy} single={!!skillId} onChange={setSelected} />}
    <Failure error={error} />
    <ImportFeedback result={result} />
    {busy && <p role="status">处理中…</p>}
    {uploadId && <button type="button" disabled={busy || !selected.length} onClick={() => void commit()}>确认收录</button>}
  </div>;
}
