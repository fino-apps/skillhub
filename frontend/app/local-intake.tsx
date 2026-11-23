import { useId, useState, type FormEvent, type ReactNode } from "react";
import { commitUpload } from "@/generated/api-client";
import { importSourceName, message, stageZip, tagsFrom, type Candidate, type ImportResult } from "./library-api";
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

export function ImportMetadataFields({ categories, existingTags, defaultSource, disabled }: {
  categories: string[]; existingTags: string[]; defaultSource: string; disabled: boolean;
}) {
  const hintId = useId();
  const categoriesId = useId();
  const tagsId = useId();
  const [summary, setSummary] = useState("");
  const [tagPrefix, setTagPrefix] = useState("");
  const enteredTags = new Set(tagsFrom(tagPrefix));
  const suggestedTags = existingTags.filter((tag) => !enteredTags.has(tag) && !/[,，\n]/.test(tag));

  return <details className="skill-disclosure file-import-metadata" onChange={(event) => {
    const form = event.currentTarget.closest("form");
    if (!form) return;
    const { category, tags, purpose, upstreamName, upstreamUrl, notes } = importMetadataValues(form);
    const classification = [category, tags.length ? `${tags.length} 个标签` : ""].filter(Boolean).join(" · ");
    setSummary(classification || (purpose || upstreamName || upstreamUrl || notes ? "已填写" : ""));
  }}>
    <summary>整理信息{summary ? <span title={summary}>{summary}</span> : "（选填）"}</summary>
    <fieldset className="skill-fields" disabled={disabled} aria-describedby={hintId}>
      <legend className="sr-only">整理信息</legend>
      <span className="skillhub-hint" id={hintId}>应用到本批新收录的 Skill</span>
      <label>短用途<input name="purpose" maxLength={240} /></label>
      <label>分类<input name="category" maxLength={128} list={categoriesId} /></label>
      <datalist id={categoriesId}>{categories.map((name) => <option key={name} value={name} />)}</datalist>
      <label>标签<input name="tags" list={tagsId} placeholder="选择或输入标签，用逗号分隔"
        onChange={(event) => setTagPrefix(event.target.value.match(/^.*[,，]\s*/)?.[0] ?? "")} /></label>
      <datalist id={tagsId}>{suggestedTags.map((tag) => <option key={tag} value={tagPrefix + tag} />)}</datalist>
      <label>来源名称<input name="upstreamName" maxLength={120} placeholder={defaultSource} /></label>
      <label>来源链接<input name="upstreamUrl" type="url" maxLength={2048} /></label>
      <label>备注<textarea name="notes" rows={3} maxLength={4096} /></label>
    </fieldset>
  </details>;
}

function importMetadataValues(form: HTMLFormElement) {
  const values = new FormData(form);
  return {
    category: String(values.get("category") ?? "").trim(),
    tags: tagsFrom(String(values.get("tags") ?? "")),
    purpose: String(values.get("purpose") ?? "").trim(),
    upstreamName: String(values.get("upstreamName") ?? "").trim(),
    upstreamUrl: String(values.get("upstreamUrl") ?? "").trim(),
    notes: String(values.get("notes") ?? ""),
  };
}

export function readImportMetadata(form: HTMLFormElement) {
  const metadata = importMetadataValues(form);
  const { tags, upstreamUrl } = metadata;
  let error = "";
  if (tags.length > 100 || tags.some((tag) => [...tag].length > 128)) {
    error = "最多填写 100 个标签，每个标签最多 128 个字符";
  } else if (upstreamUrl && (!/^https?:\/\//i.test(upstreamUrl) || !URL.canParse(upstreamUrl))) {
    error = "请输入以 http:// 或 https:// 开头的来源链接";
  }
  if (error) {
    const details = form.querySelector<HTMLDetailsElement>(".file-import-metadata");
    if (details) details.open = true;
    throw new Error(error);
  }
  return metadata;
}

function directoryInput(input: HTMLInputElement | null) {
  input?.setAttribute("webkitdirectory", "");
}

async function archiveDirectory(files: FileList) {
  const entries = Array.from(files).filter((file) => !file.webkitRelativePath.split("/").slice(1).includes(".git"));
  if (!entries.some((file) => file.name === "SKILL.md")) {
    throw new Error("所选目录中没有找到 SKILL.md");
  }
  const sizeLimit = 64 * 1024 * 1024;
  if (entries.length > 10000 || entries.reduce((total, file) => total + file.size, 0) > sizeLimit) {
    throw new Error("所选目录最多支持 10000 个文件，文件总大小不能超过 64 MiB");
  }

  const { zip } = await import("fflate");
  const contents = Object.fromEntries(await Promise.all(entries.map(async (file) =>
    [file.webkitRelativePath, new Uint8Array(await file.arrayBuffer())] as const,
  )));
  const archive = await new Promise<Uint8Array>((resolve, reject) => {
    zip(contents, (error, data) => error ? reject(error) : resolve(data));
  });
  if (archive.byteLength > sizeLimit) {
    throw new Error("打包后超过 64 MiB，请选择更小的目录");
  }
  const root = entries[0].webkitRelativePath.split("/")[0];
  return new File([archive], `${root}.zip`, { type: "application/zip" });
}

export function UploadIntake({ skillId, directory = false, categories = [], existingTags = [], onComplete, children }: {
  skillId?: string; directory?: boolean; categories?: string[]; existingTags?: string[]; onComplete: (result: ImportResult, pending?: boolean) => void; children?: ReactNode;
}) {
  const [uploadId, setUploadId] = useState("");
  const [items, setItems] = useState<Candidate[]>([]);
  const [selected, setSelected] = useState<string[]>([]);
  const [busy, setBusy] = useState(false);
  const [error, setError] = useState("");
  const [result, setResult] = useState<ImportResult | null>(null);

  async function read(input: HTMLInputElement) {
    const files = input.files;
    if (!files?.length) return;
    setBusy(true);
    setError("");
    setResult(null);
    setItems([]);
    setUploadId("");
    try {
      const file = directory ? await archiveDirectory(files) : files[0];
      const reply = await stageZip(file);
      const candidates = reply.skills ?? [];
      setUploadId(reply.uploadId);
      setItems(candidates);
      setSelected(candidates.slice(0, skillId ? 1 : importLimit).map((item) => item.path ?? ""));
    } catch (err) {
      setError(message(err));
      input.value = "";
    } finally {
      setBusy(false);
    }
  }

  async function commit(event: FormEvent<HTMLFormElement>) {
    event.preventDefault();
    if (busy || !uploadId || !selected.length) return;
    const form = event.currentTarget;
    setBusy(true);
    setError("");
    try {
      const metadata = skillId ? {} : readImportMetadata(form);
      const reply = await commitUpload<ImportResult>({ uploadId, skillPaths: selected, targetSkillId: skillId, directoryUpload: directory, ...metadata });
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
    <label>{directory ? "选择本机 Skill 目录" : "ZIP 文件"}
      <input type="file" accept={directory ? undefined : ".zip,application/zip"} disabled={busy}
        ref={directory ? directoryInput : undefined} onChange={(event) => void read(event.currentTarget)} />
    </label>
    <Failure error={error} />
    <ImportFeedback result={result} />
    {busy && !uploadId && <p role="status">正在读取{directory ? "目录" : "ZIP 文件"}…</p>}
    {items.length > 0 && <form key={uploadId} className="github-form" onSubmit={commit}>
      <CandidatePicker items={items} selected={selected} disabled={busy} single={!!skillId} onChange={setSelected} />
      {!skillId && <ImportMetadataFields categories={categories} existingTags={existingTags}
        defaultSource={importSourceName(directory ? "directory_upload" : "zip")} disabled={busy} />}
      <button type="submit" disabled={busy || !selected.length}>
        {busy ? "收录中…" : skillId ? "确认收录" : `收录 ${selected.length} 项`}
      </button>
    </form>}
    {!busy && children && <div className="file-import-options">{children}</div>}
  </div>;
}
