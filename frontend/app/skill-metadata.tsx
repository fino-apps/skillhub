import { useState, type FormEvent } from "react";
import { batchUpdateSkillMetadata, updateSkill } from "@/generated/api-client";
import { message, tagsFrom, type Skill } from "./library-api";
import { Failure } from "./skill-ui";

export default function SkillMetadata({ skill, categories, onSaved }: { skill: Skill; categories: string[]; onSaved: () => void }) {
  const [busy, setBusy] = useState(false);
  const [error, setError] = useState("");

  async function save(event: FormEvent<HTMLFormElement>) {
    event.preventDefault();
    const values = Object.fromEntries(new FormData(event.currentTarget));
    const url = String(values.upstreamUrl ?? "").trim();
    if (url && (!/^https?:\/\//i.test(url) || !URL.canParse(url))) {
      setError("请输入以 http:// 或 https:// 开头的来源链接");
      return;
    }
    setBusy(true);
    setError("");
    try {
      await updateSkill({ skillId: skill.id, skill: { ...values, upstreamName: skill.upstreamName, tags: tagsFrom(String(values.tags ?? "")) } });
      onSaved();
    } catch (err) {
      setError(message(err));
    } finally {
      setBusy(false);
    }
  }

  return <form className="github-form" onSubmit={save}>
    <fieldset disabled={busy} className="skill-fields">
      <label>短用途<input name="purpose" defaultValue={skill.purpose} maxLength={240} /></label>
      <label>分类<input name="category" defaultValue={skill.category} maxLength={128} list="existing-categories" /></label>
      <datalist id="existing-categories">{categories.map((name) => <option key={name} value={name} />)}</datalist>
      <label>标签<input name="tags" defaultValue={skill.tags?.join(", ")} placeholder="用逗号分隔" /></label>
      <label>来源链接<input name="upstreamUrl" defaultValue={skill.upstreamUrl} type="url" maxLength={2048} /></label>
      <label>备注<textarea name="notes" defaultValue={skill.notes} rows={3} maxLength={4096} /></label>
    </fieldset>
    <Failure error={error} />
    <button type="submit" disabled={busy}>{busy ? "保存中…" : "保存"}</button>
  </form>;
}

export function BatchMetadata({ ids, categories, onSaved }: { ids: string[]; categories: string[]; onSaved: () => void }) {
  const [busy, setBusy] = useState(false);
  const [error, setError] = useState("");
  const [categoryMode, setCategoryMode] = useState("keep");

  async function save(event: FormEvent<HTMLFormElement>) {
    event.preventDefault();
    const values = new FormData(event.currentTarget);
    const category = categoryMode === "keep" ? undefined : String(values.get("category") ?? "");
    const addTags = tagsFrom(String(values.get("addTags") ?? ""));
    const removeTags = tagsFrom(String(values.get("removeTags") ?? ""));
    if (category === undefined && !addTags.length && !removeTags.length) {
      setError("请填写要修改的分类或标签");
      return;
    }
    setBusy(true);
    setError("");
    try {
      await batchUpdateSkillMetadata({ changes: ids.map((skillId) => ({ skillId, category, addTags, removeTags })) });
      onSaved();
    } catch (err) {
      setError(message(err));
    } finally {
      setBusy(false);
    }
  }

  return <form className="github-form" onSubmit={save}>
    <fieldset disabled={busy} className="skill-fields">
      <label>分类<select value={categoryMode} onChange={(event) => setCategoryMode(event.target.value)}>
        <option value="keep">保持原值</option><option value="set">设为指定分类</option><option value="clear">设为未分类</option>
      </select></label>
      {categoryMode === "set" && <label>分类名称<input name="category" list="batch-categories" maxLength={128} required /></label>}
      <datalist id="batch-categories">{categories.map((name) => <option key={name} value={name} />)}</datalist>
      <label>添加标签<input name="addTags" placeholder="用逗号分隔" /></label>
      <label>移除标签<input name="removeTags" placeholder="用逗号分隔" /></label>
    </fieldset>
    <Failure error={error} />
    <button type="submit" disabled={busy}>{busy ? "保存中…" : `应用到 ${ids.length} 项`}</button>
  </form>;
}
