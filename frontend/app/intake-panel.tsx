import { useState } from "react";
import { importSkill, previewImport } from "@/generated/api-client";

type Candidate = { path: string; name: string };
type FileInfo = { path: string; sensitive?: boolean };
type Preview = { candidate: Candidate; files: FileInfo[]; selected: string[]; digest: string; error?: string; saved?: boolean };

type Props = { basePath: string; candidates: Candidate[]; onComplete: () => void };

function fullPath(base: string, path: string) {
  return path === "." ? base : `${base.replace(/\/+$/, "")}/${path}`;
}

function errorText(error: unknown) {
  return error instanceof Error && error.message ? error.message : "操作失败";
}

export default function IntakePanel({ basePath, candidates, onComplete }: Props) {
  const [checked, setChecked] = useState<string[]>([]);
  const [previews, setPreviews] = useState<Preview[]>([]);
  const [busy, setBusy] = useState(false);

  function toggle(path: string) {
    setChecked((value) => value.includes(path) ? value.filter((item) => item !== path) : [...value, path]);
    setPreviews([]);
  }

  async function prepare() {
    setBusy(true);
    const results: Preview[] = [];
    for (const candidate of candidates.filter((item) => checked.includes(item.path))) {
      try {
        const result = await previewImport<{ files?: FileInfo[]; sourceDigest: string }>({ source: { localPath: fullPath(basePath, candidate.path) } });
        const files = result.files ?? [];
        results.push({ candidate, files, digest: result.sourceDigest, selected: files.filter((file) => !file.sensitive).map((file) => file.path) });
      } catch (error) {
        results.push({ candidate, files: [], digest: "", selected: [], error: errorText(error) });
      }
    }
    setPreviews(results);
    setBusy(false);
  }

  function toggleFile(candidatePath: string, filePath: string) {
    setPreviews((items) => items.map((item) => item.candidate.path !== candidatePath ? item : {
      ...item, selected: item.selected.includes(filePath) ? item.selected.filter((path) => path !== filePath) : [...item.selected, filePath],
    }));
  }

  async function save() {
    setBusy(true);
    const results = [...previews];
    for (let index = 0; index < results.length; index++) {
      const item = results[index];
      if (item.saved || item.error || !item.selected.includes("SKILL.md")) continue;
      try {
        await importSkill({ source: { localPath: fullPath(basePath, item.candidate.path) }, expectedSourceDigest: item.digest, includedFiles: item.selected, version: { version: "1.0.0" } });
        results[index] = { ...item, saved: true };
      } catch (error) {
        const detail = errorText(error);
        results[index] = { ...item, error: detail === "resource conflict" ? "管理库已有同名 Skill，请先检查已有条目。" : detail };
      }
      setPreviews([...results]);
    }
    setBusy(false);
    onComplete();
  }

  return <div className="intake-panel">
    <div className="intake-header"><strong>选择 Skill</strong><button type="button" className="secondary" onClick={() => void prepare()} disabled={busy || checked.length === 0}>{busy ? "处理中…" : `预览已选 (${checked.length})`}</button></div>
    <div className="intake-choices">{candidates.map((candidate) => <label key={candidate.path}><input type="checkbox" checked={checked.includes(candidate.path)} onChange={() => toggle(candidate.path)} disabled={busy} />{candidate.name}</label>)}</div>
    {previews.length > 0 && <div className="intake-preview"><h3>检查文件</h3><p>疑似敏感文件默认不收录。</p>{previews.map((item) => <section key={item.candidate.path} className="intake-item"><strong>{item.candidate.name}</strong>{item.saved ? <p className="intake-success">已收录</p> : item.error ? <p className="intake-error">{item.error}</p> : <div className="intake-files">{item.files.map((file) => <label key={file.path}><input type="checkbox" checked={item.selected.includes(file.path)} disabled={busy || file.path === "SKILL.md"} onChange={() => toggleFile(item.candidate.path, file.path)} /><span>{file.path}</span>{file.sensitive && <em>疑似敏感</em>}</label>)}</div>}</section>)}<button type="button" onClick={() => void save()} disabled={busy || !previews.some((item) => !item.error && !item.saved)}>{busy ? "收录中…" : "确认收录"}</button></div>}
  </div>;
}
