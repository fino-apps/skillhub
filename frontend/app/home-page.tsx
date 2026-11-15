import { useEffect, useState, type FormEvent } from "react";
import { listSkills, scanProject } from "@/generated/api-client";
import "./skillhub.css";

type Skill = { id: string; name: string; description: string; category?: string };
type ProjectSkill = { path: string; name: string; description: string };

export default function HomePage() {
  const [skills, setSkills] = useState<Skill[]>([]);
  const [projectSkills, setProjectSkills] = useState<ProjectSkill[] | null>(null);
  const [projectPath, setProjectPath] = useState("");
  const [keyword, setKeyword] = useState("");
  const [loading, setLoading] = useState(true);
  const [scanning, setScanning] = useState(false);
  const [error, setError] = useState("");

  useEffect(() => {
    const controller = new AbortController();
    listSkills<{ skills?: Skill[] }>({}, { signal: controller.signal })
      .then((response) => setSkills(response.skills ?? []))
      .catch((reason: unknown) => { if (!controller.signal.aborted) setError(reason instanceof Error ? reason.message : "无法加载 Skill"); })
      .finally(() => { if (!controller.signal.aborted) setLoading(false); });
    return () => controller.abort();
  }, []);

  async function handleScan(event: FormEvent) {
    event.preventDefault();
    if (!projectPath.trim()) return;
    setScanning(true);
    setError("");
    try {
      const response = await scanProject<{ skills?: ProjectSkill[] }>({ projectPath: projectPath.trim() });
      setProjectSkills(response.skills ?? []);
    } catch (reason) {
      setError(reason instanceof Error ? reason.message : "扫描失败，请检查项目路径");
    } finally {
      setScanning(false);
    }
  }

  const filtered = skills.filter((skill) => `${skill.name} ${skill.description} ${skill.category ?? ""}`.toLocaleLowerCase().includes(keyword.toLocaleLowerCase()));

  return (
    <main className="skillhub-shell">
      <header className="skillhub-heading">
        <div><h1>Skill 库</h1><p>浏览管理库，并发现项目中的 Skill。</p></div>
      </header>
      {error && <p role="alert" className="skillhub-error">{error}</p>}
      <section className="skillhub-section" aria-labelledby="library-title">
        <div className="skillhub-section-heading"><h2 id="library-title">管理库</h2><label>搜索 <input value={keyword} onChange={(event) => setKeyword(event.target.value)} placeholder="名称、描述或分类" /></label></div>
        {loading ? <p aria-live="polite">正在加载…</p> : filtered.length === 0 ? <p className="skillhub-empty">{skills.length ? "没有符合搜索条件的 Skill。" : "管理库中还没有 Skill。"}</p> : (
          <ul className="skillhub-list">{filtered.map((skill) => <li key={skill.id}><strong>{skill.name}</strong><p>{skill.description}</p>{skill.category && <span>{skill.category}</span>}</li>)}</ul>
        )}
      </section>
      <section className="skillhub-section" aria-labelledby="project-title">
        <h2 id="project-title">发现项目 Skill</h2>
        <p>输入本机项目的绝对路径，查看其中的 Skill。</p>
        <form className="skillhub-form" onSubmit={handleScan}>
          <label htmlFor="project-path">项目路径</label>
          <div><input id="project-path" value={projectPath} onChange={(event) => setProjectPath(event.target.value)} placeholder="/Users/you/project" required /><button type="submit" disabled={scanning}>{scanning ? "扫描中…" : "扫描项目"}</button></div>
        </form>
        {projectSkills && (projectSkills.length ? <ul className="skillhub-list">{projectSkills.map((skill) => <li key={skill.path}><strong>{skill.name}</strong><span>{skill.path}</span><p>{skill.description}</p></li>)}</ul> : <p className="skillhub-empty">没有发现 Skill，请检查项目路径。</p>)}
      </section>
    </main>
  );
}
