import { useState, type FormEvent } from "react";
import { compareSkills, createVersion, previewImport } from "@/generated/api-client";
import { CompareResults, type Change } from "./compare-panel";
import { backendDirectoryLabel, canPickDirectory, message, pickDirectory, type Skill } from "./library-api";
import { Failure } from "./skill-ui";
import type { SkillVersion } from "./skill-version";

type Preview = { commit?: string; digest?: string; changes?: Change[] };

export default function VersionPanel({ skill, from, path: initialPath, onSaved }: {
  skill: Skill; from: "github" | "folder"; path?: string; onSaved: (version: SkillVersion) => void;
}) {
  const [ref, setRef] = useState(skill.defaultVersion?.sourceRef ?? "");
  const [path, setPath] = useState(initialPath || skill.localPath || "");
  const [notes, setNotes] = useState("");
  const [makeDefault, setDefault] = useState(false);
  const [preview, setPreview] = useState<Preview | null>(null);
  const [busy, setBusy] = useState("");
  const [error, setError] = useState("");

  async function run(key: string, action: () => Promise<void>) {
    setBusy(key);
    setError("");
    try {
      await action();
    } catch (err) {
      setError(message(err));
    } finally {
      setBusy("");
    }
  }

  function inspect(event: FormEvent<HTMLFormElement>) {
    event.preventDefault();
    setPreview(null);
    const version = skill.defaultVersion;
    void run("preview", async () => {
      if (from === "folder") {
        if (!version?.available) {
          setPreview({});
          return;
        }
        const comparison = await compareSkills<{ rightDigest: string; files?: Change[] }>({
          left: { versionId: version.id }, right: { localPath: path },
        });
        setPreview({ digest: comparison.rightDigest, changes: comparison.files ?? [] });
        return;
      }
      const source = { repositoryUrl: skill.repositoryUrl, skillPath: skill.repositoryPath ?? "", ref };
      const { resolvedCommitSha: commit } = await previewImport<{ resolvedCommitSha: string }>({ source });
      if (!version?.available || version.commitSha === commit) {
        setPreview({ commit });
        return;
      }
      const reply = await compareSkills<{ files?: Change[] }>({
        left: { versionId: version.id }, right: { github: { ...source, ref: commit } },
      });
      setPreview({ commit, changes: reply.files ?? [] });
    });
  }

  function save() {
    void run("save", async () => {
      const result = await createVersion<{ version: SkillVersion }>({
        skillId: skill.id,
        version: {
          notes, makeDefault,
          localPath: from === "folder" ? path : undefined,
          expectedDigest: preview?.digest, commitSha: preview?.commit, sourceRef: from === "github" ? ref : undefined,
        },
      });
      onSaved(result.version);
    });
  }

  return <>
    <form className="github-form" onSubmit={inspect}>
      <fieldset className="skill-fields" disabled={!!busy}>
        {from === "github"
          ? <label>分支、标签或 commit<input value={ref} maxLength={200} placeholder="仓库默认分支"
            onChange={(event) => { setRef(event.target.value); setPreview(null); }} /></label>
          : <><label>{backendDirectoryLabel}<input value={path} onChange={(event) => { setPath(event.target.value); setPreview(null); }} required /></label>
            {canPickDirectory && <button type="button" className="secondary" onClick={() => void run("choose", async () => {
              const selected = await pickDirectory();
              if (selected) { setPath(selected); setPreview(null); }
            })}>选择文件夹</button>}</>}
      </fieldset>
      <button className="secondary" type="submit" disabled={!!busy}>{busy === "preview" ? "读取中…" : "预览差异"}</button>
    </form>
    <Failure error={error} />
    {preview && <div className="github-preview">
      {preview.commit && <p>Commit <code>{preview.commit}</code></p>}
      {preview.changes && <CompareResults files={preview.changes} />}
      {preview.commit && preview.commit === skill.defaultVersion?.commitSha && <p>此 commit 已收录。</p>}
      <fieldset className="skill-fields" disabled={!!busy}>
        <label>版本备注<input value={notes} maxLength={4096} onChange={(event) => setNotes(event.target.value)} /></label>
        <label className="github-checkbox"><input type="checkbox" checked={makeDefault}
          onChange={(event) => setDefault(event.target.checked)} />同时设为默认</label>
      </fieldset>
      <button type="button" disabled={!!busy} onClick={save}>{busy === "save" ? "保存中…" : "保存版本"}</button>
    </div>}
  </>;
}
