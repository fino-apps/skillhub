import { useEffect, useState } from "react";
import { batchUpdateSkillMetadata, createOrganizationTask, getOrganizationTask } from "@/generated/api-client";
import { message, tagsFrom, type Skill } from "./library-api";
import { Failure } from "./skill-ui";

type Change = { skillId: string; revision?: string; purpose?: string; category?: string; addTags?: string[] };
type Suggestion = { name: string; change?: Change; error?: string };
type Task = { id: string; total?: number; results?: Suggestion[]; done?: boolean };

export default function OrganizePanel({ skills, uncategorized, categories, onSaved }: {
  skills: Skill[]; uncategorized: boolean; categories: string[]; onSaved: () => void;
}) {
  const [task, setTask] = useState<Task | null>(null);
  const [pending, setPending] = useState(false);
  const [error, setError] = useState("");
  const [retained, setRetained] = useState<Suggestion[]>([]);
  const [edits, setEdits] = useState<Record<string, Change>>({});
  const [excluded, setExcluded] = useState<string[]>([]);
  const taskId = task?.id;
  const done = task?.done;

  useEffect(() => {
    if (!taskId || done) return;
    const controller = new AbortController();
    let timer: ReturnType<typeof setTimeout>;
    async function poll() {
      try {
        const state = await getOrganizationTask<Task>({ taskId }, { signal: controller.signal });
        if (!controller.signal.aborted) {
          setTask(state);
          setError("");
          if (!state.done) timer = setTimeout(() => void poll(), 900);
        }
      } catch (err) {
        if (!controller.signal.aborted) {
          setError(message(err));
          timer = setTimeout(() => void poll(), 1800);
        }
      }
    }
    timer = setTimeout(() => void poll(), 600);
    return () => { controller.abort(); clearTimeout(timer); };
  }, [taskId, done]);

  async function start(ids = skills.map(({ id }) => id), keep: Suggestion[] = []) {
    setPending(true);
    setError("");
    try {
      setTask(await createOrganizationTask<Task>({ skillIds: ids, uncategorized: uncategorized && !ids.length }));
      setRetained(keep);
      if (!keep.length) { setEdits({}); setExcluded([]); }
    } catch (err) {
      setError(message(err));
    } finally {
      setPending(false);
    }
  }

  const currentResults = task?.results ?? [];
  const results = [...retained, ...currentResults];
  const eligible = results
    .filter((item) => !item.error && item.change && !excluded.includes(item.change.skillId))
    .map(({ change }) => edits[change!.skillId] ?? change!)
    .filter((change) => change.purpose || change.category || change.addTags?.length)
    .map((change) => ({
      ...change, purpose: change.purpose || undefined, category: change.category || undefined,
    }));
  const failed = results.filter((item) => item.error && item.change).map((item) => item.change!.skillId);

  async function apply() {
    setPending(true);
    setError("");
    try {
      await batchUpdateSkillMetadata({ changes: eligible });
      onSaved();
    } catch (err) {
      setError(message(err));
    } finally {
      setPending(false);
    }
  }

  return <div className="skill-fields">
    <Failure error={error} />
    {!task && <>
      <p>{uncategorized && !skills.length ? "处理未分类条目，每批最多 100 项。" : `处理选中的 ${skills.length} 项。`}</p>
      <button type="button" disabled={pending} onClick={() => void start()}>{pending ? "启动中…" : "生成建议"}</button>
    </>}
    {task && <p role="status">{task.done ? "建议已生成" : "正在整理"} · {currentResults.length}/{task.total ?? 0}</p>}
    {task?.done && results.length > 0 && <div className="organization-results">
      {results.map((item) => {
        if (!item.change) return null;
        const id = item.change.skillId;
        const value = edits[id] ?? item.change;
        const skill = skills.find((record) => record.id === id);
        function change(edit: Partial<Change>) {
          setEdits((previous) => ({ ...previous, [id]: { ...value, ...edit } }));
        }
        return <section key={id} className="skill-fields">
          <label className="github-checkbox"><input type="checkbox" disabled={!!item.error || pending}
            checked={!item.error && !excluded.includes(id)} onChange={() => {
              setExcluded((old) => old.includes(id) ? old.filter((value) => value !== id) : [...old, id]);
            }} /><strong>{item.name}</strong></label>
          {item.error ? <Failure error={item.error} /> : <fieldset className="skill-fields" disabled={pending || excluded.includes(id)}>
            {value.purpose !== undefined && <label>用途 <small>{skill?.purpose ? `${skill.purpose} →` : ""}</small>
              <input maxLength={240} value={value.purpose} onChange={(event) => change({ purpose: event.target.value })} /></label>}
            {value.category !== undefined && <label>分类 <small>{categories.includes(value.category) ? "" : "新分类"}</small>
              <input maxLength={128} value={value.category} onChange={(event) => change({ category: event.target.value })} /></label>}
            {item.change.addTags !== undefined && item.change.addTags.length > 0 && <label>标签
              <input defaultValue={value.addTags?.join(", ")} onChange={(event) => change({ addTags: tagsFrom(event.target.value) })} /></label>}
            {!value.purpose && !value.category && !value.addTags?.length && <span className="skillhub-hint">没有建议</span>}
          </fieldset>}
        </section>;
      })}
    </div>}
    {task?.done && <div className="library-actions">
      <button type="button" disabled={pending || !eligible.length} onClick={() => void apply()}>应用选中建议（{eligible.length}）</button>
      {failed.length > 0 && <button type="button" className="secondary" disabled={pending} onClick={() => void start(failed, results.filter((result) => !result.error))}>重试失败项</button>}
    </div>}
  </div>;
}
