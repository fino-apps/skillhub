import { useState, type FormEvent } from "react";
import { importSkill, previewImport } from "@/generated/api-client";
import { message, type Candidate, type ImportResult } from "./library-api";
import { CandidatePicker, ImportFeedback, ImportMetadataFields, readImportMetadata } from "./local-intake";
import { Failure } from "./skill-ui";

function githubLocation(link: string, overrideRef: string, overridePath: string) {
  const url = new URL(link);
  const [owner, repository, route, ...rest] = url.pathname.split("/").filter(Boolean);
  if (url.protocol !== "https:" || url.hostname !== "github.com" || !owner || !repository || (route && route !== "tree")) {
    throw new Error("请输入 GitHub 仓库或目录链接");
  }
  return {
    repositoryUrl: `https://github.com/${owner}/${repository}`,
    ref: overrideRef || (route === "tree" ? decodeURIComponent(rest[0] || "") : ""),
    skillPath: overridePath || (route === "tree" ? rest.slice(1).map(decodeURIComponent).join("/") : ""),
  };
}

type Preview = {
  resolvedCommitSha: string;
  skills?: Candidate[];
};

export default function GitHubIntake({ categories, onComplete }: { categories: string[]; onComplete: (result: ImportResult) => void }) {
  const [preview, setPreview] = useState<{ source: ReturnType<typeof githubLocation>; result: Preview } | null>(null);
  const [selected, setSelected] = useState<string[]>([]);
  const [busy, setBusy] = useState(false);
  const [error, setError] = useState("");
  const [feedback, setFeedback] = useState<ImportResult | null>(null);

  async function inspect(event: FormEvent<HTMLFormElement>) {
    event.preventDefault();
    const values = new FormData(event.currentTarget);
    setPreview(null);
    setFeedback(null);
    setBusy(true);
    setError("");
    try {
      const source = githubLocation(String(values.get("link")).trim(), String(values.get("ref")).trim(), String(values.get("path")).trim());
      const result = await previewImport<Preview>({ source });
      setPreview({ source, result });
      setSelected((result.skills ?? []).slice(0, 100).map((skill) => skill.path ?? ""));
    } catch (err) {
      setError(message(err));
    } finally {
      setBusy(false);
    }
  }

  async function save(event: FormEvent<HTMLFormElement>) {
    event.preventDefault();
    if (!preview || !selected.length || busy) return;
    const form = event.currentTarget;
    setBusy(true);
    setError("");
    try {
      const metadata = readImportMetadata(form);
      const result = await importSkill<ImportResult>({
        repositoryUrl: preview.source.repositoryUrl, sourceRef: preview.source.ref,
        commitSha: preview.result.resolvedCommitSha, skillPaths: selected,
        ...metadata,
      });
      if (result.failures?.length) {
        setFeedback(result);
        setSelected(result.failures.map(({ path }) => path ?? ""));
      }
      onComplete(result);
    } catch (err) {
      setError(message(err));
    } finally {
      setBusy(false);
    }
  }

  return <>
    <form className="github-form" onSubmit={inspect} onChange={() => { setPreview(null); setFeedback(null); }}>
      <fieldset className="skill-fields" disabled={busy}>
        <label>仓库或 Skill 目录链接<input name="link" type="url" placeholder="https://github.com/owner/repo" required /></label>
        <details className="skill-disclosure">
          <summary>指定版本或仓库路径</summary>
          <div className="skill-fields">
            <label>分支、标签或 commit<input name="ref" maxLength={200} /></label>
            <label>仓库内路径<input name="path" placeholder="skills/example" maxLength={1024} /></label>
          </div>
        </details>
      </fieldset>
      <button className="secondary" type="submit" disabled={busy}>{busy && !preview ? "扫描中…" : "扫描技能"}</button>
    </form>
    <Failure error={error} />
    <ImportFeedback result={feedback} />
    {preview && <div className="github-preview">
      <p>Commit <code>{preview.result.resolvedCommitSha}</code></p>
      {(preview.result.skills?.length ?? 0) > 0
        ? <form className="github-form" onSubmit={save}>
          <CandidatePicker items={preview.result.skills ?? []} selected={selected} onChange={setSelected} disabled={busy} />
          <ImportMetadataFields categories={categories} disabled={busy} />
          <button type="submit" disabled={busy || !selected.length}>
            {busy ? "收录中…" : `收录所选（${selected.length}）`}
          </button>
        </form>
        : <p role="status">没有找到 Skill</p>}
    </div>}
  </>;
}
