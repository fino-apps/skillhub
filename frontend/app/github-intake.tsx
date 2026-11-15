import { useState, type FormEvent } from "react";
import { importSkill, previewImport } from "@/generated/api-client";

type GitHub = { repositoryUrl: string; skillPath: string; ref: string };
type Preview = { sourceDigest: string; resolvedCommitSha: string; files?: { path: string; sensitive?: boolean }[] };

export default function GitHubIntake({ onComplete }: { onComplete: () => void }) {
  const [source, setSource] = useState<GitHub>({ repositoryUrl: "", skillPath: "", ref: "main" });
  const [preview, setPreview] = useState<Preview | null>(null);
  const [busy, setBusy] = useState(false);
  const [feedback, setFeedback] = useState("");

  function change(field: keyof GitHub, value: string) {
    setSource((current) => ({ ...current, [field]: value }));
    setPreview(null);
    setFeedback("");
  }

  async function inspect(event: FormEvent<HTMLFormElement>) {
    event.preventDefault();
    setBusy(true);
    setFeedback("");
    setPreview(null);
    try {
      const result = await previewImport<Preview>({ source: { github: source } });
      setPreview(result);
    } catch (error) {
      setFeedback(error instanceof Error && error.message !== "bad request" ? error.message : "无法读取来源，请检查仓库、路径、版本及网络。");
    } finally { setBusy(false); }
  }

  async function save() {
    if (!preview) return;
    setBusy(true);
    setFeedback("");
    try {
      await importSkill({ source: { github: source }, expectedSourceDigest: preview.sourceDigest, includedFiles: ["SKILL.md"], version: { version: "1.0.0", sourceRef: source.ref, commitSha: preview.resolvedCommitSha } });
      setFeedback("已收录公开来源。");
      setPreview(null);
      onComplete();
    } catch (error) {
      const detail = error instanceof Error ? error.message : "收录失败";
      setFeedback(detail === "resource conflict" ? "此来源已收录，请查看管理库。" : detail === "bad request" ? "来源发生变化或不可访问，请重新预览。" : detail);
    } finally { setBusy(false); }
  }

  return <div className="github-intake">
    <h2>收录公开 Skill</h2>
    <p>填写 GitHub 仓库和 Skill 路径；仅保存来源及固定版本。</p>
    <form onSubmit={inspect} className="github-form">
      <label>仓库地址<input type="url" disabled={busy} value={source.repositoryUrl} onChange={(event) => change("repositoryUrl", event.target.value)} placeholder="https://github.com/owner/repo" required /></label>
      <div><label>仓库内路径<input disabled={busy} value={source.skillPath} onChange={(event) => change("skillPath", event.target.value)} placeholder="skills/example（根目录可留空）" /></label><label>分支或标签<input disabled={busy} value={source.ref} onChange={(event) => change("ref", event.target.value)} required /></label></div>
      <button className="secondary" type="submit" disabled={busy}>{busy ? "读取中…" : "预览来源"}</button>
    </form>
    {feedback && <p role="status" className="skillhub-hint">{feedback}</p>}
    {preview && <div className="github-preview"><strong>来源预览</strong><p>固定 commit：<code>{preview.resolvedCommitSha}</code></p><p>{preview.files?.length ?? 0} 个文件 · 收录前请检查来源和许可</p><details><summary>查看文件列表</summary><ul>{preview.files?.map((file) => <li key={file.path}>{file.path}{file.sensitive ? "（疑似敏感）" : ""}</li>)}</ul></details><button type="button" onClick={() => void save()} disabled={busy}>{busy ? "收录中…" : "收录公开来源"}</button></div>}
  </div>;
}
