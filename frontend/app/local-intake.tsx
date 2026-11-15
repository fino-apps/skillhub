import { useState, type FormEvent } from "react";
import { importSkill, scanProject } from "@/generated/api-client";

type Candidate = { path: string; name: string; description: string };

export default function LocalIntake({ onComplete }: { onComplete: () => void }) {
  const [path, setPath] = useState("");
  const [candidates, setCandidates] = useState<Candidate[] | null>(null);
  const [selected, setSelected] = useState<string[]>([]);
  const [pending, setPending] = useState<"scan" | "save" | null>(null);
  const [feedback, setFeedback] = useState("");
  const [error, setError] = useState("");

  async function scan(event: FormEvent<HTMLFormElement>) {
    event.preventDefault();
    const root = path.trim();
    if (!root) return;
    setPending("scan");
    setCandidates(null);
    setFeedback("");
    setError("");
    try {
      const result = await scanProject<{ skills?: Candidate[] }>({ projectPath: root });
      const found = (result.skills ?? []).map((skill) => ({
        ...skill,
        path: skill.path === "." ? root : (root.endsWith("/") ? root : root + "/") + skill.path,
      }));
      setCandidates(found);
      setSelected(found.slice(0, 100).map((skill) => skill.path));
    } catch {
      setError("无法扫描。请输入本机项目或 Skill 目录的绝对路径。");
    } finally {
      setPending(null);
    }
  }

  async function save() {
    setPending("save");
    setFeedback("");
    setError("");
    try {
      const result = await importSkill<{ addedCount?: number; existingCount?: number }>({ skillPaths: selected });
      setCandidates(null);
      setSelected([]);
      setFeedback(`已保存 ${result.addedCount ?? 0} 个 Skill 的初始版本，跳过 ${result.existingCount ?? 0} 个已收录目录。`);
      onComplete();
    } catch {
      setError("收录失败。请检查目录是否仍可读取，且 Skill 内的文件可复制。");
    } finally {
      setPending(null);
    }
  }

  return <div>
    <form className="github-form" onSubmit={scan}>
      <label>本机项目或 Skill 目录<input value={path} maxLength={2048} onChange={(event) => {
        setPath(event.target.value);
        setCandidates(null);
        setSelected([]);
        setFeedback("");
        setError("");
      }} placeholder="/path/to/project" required disabled={!!pending} /></label>
      <button className="secondary" type="submit" disabled={!!pending}>{pending === "scan" ? "扫描中…" : "扫描目录"}</button>
    </form>
    {error && <p className="intake-error" role="alert">{error}</p>}
    {feedback && <p className="skillhub-hint" role="status">{feedback}</p>}
    {candidates && <div className="github-preview">
      {candidates.length ? <>
        <ul className="github-candidates">{candidates.map((candidate) => {
          const checked = selected.includes(candidate.path);
          return <li key={candidate.path}><label className="github-checkbox">
            <input type="checkbox" checked={checked} disabled={!!pending || (!checked && selected.length >= 100)}
              onChange={() => setSelected((items) => checked ? items.filter((item) => item !== candidate.path) : [...items, candidate.path])} />
            <span><strong>{candidate.name}</strong> <small>{candidate.path}</small><span className="candidate-description">{candidate.description}</span></span>
          </label></li>;
        })}</ul>
        {candidates.length > 100 && <p className="skillhub-hint">每次最多收录 100 个。</p>}
        <button type="button" disabled={!selected.length || !!pending} onClick={() => void save()}>{pending === "save" ? "保存中…" : `收录并保存版本（${selected.length}）`}</button>
      </> : <p role="status">没有找到 Skill。</p>}
    </div>}
  </div>;
}
