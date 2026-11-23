import { useId, useState, type FormEvent } from "react";
import { batchUpdateSkillMetadata, updateSkill } from "@/generated/api-client";
import { importSourceName, message, tagsFrom, type Skill } from "./library-api";
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
      await updateSkill({ skillId: skill.id, skill: { ...values, tags: tagsFrom(String(values.tags ?? "")) } });
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
      <label>来源名称<input name="upstreamName" defaultValue={skill.upstreamName}
        placeholder={importSourceName(skill.importKind)} maxLength={120} /></label>
      <label>来源链接<input name="upstreamUrl" defaultValue={skill.upstreamUrl} type="url" maxLength={2048} /></label>
      <label>分类<input name="category" defaultValue={skill.category} maxLength={128} list="existing-categories" /></label>
      <datalist id="existing-categories">{categories.map((name) => <option key={name} value={name} />)}</datalist>
      <label>标签<input name="tags" defaultValue={skill.tags?.join(", ")} placeholder="用逗号分隔" /></label>
      <label>备注<textarea name="notes" defaultValue={skill.notes} rows={3} maxLength={4096} /></label>
    </fieldset>
    <Failure error={error} />
    <button type="submit" disabled={busy}>{busy ? "保存中…" : "保存"}</button>
  </form>;
}

function tagLimitError(tags: string[]) {
  return tags.length > 100 || tags.some((tag) => [...tag].length > 128)
    ? "最多填写 100 个标签，每个标签最多 128 个字符"
    : "";
}

export function BatchMetadata({ skills, categories, onSaved }: { skills: Skill[]; categories: string[]; onSaved: () => void }) {
  const [busy, setBusy] = useState(false);
  const [error, setError] = useState("");
  const [categoryMode, setCategoryMode] = useState("keep");
  const [categoryName, setCategoryName] = useState("");
  const [addText, setAddText] = useState("");
  const [removeText, setRemoveText] = useState("");
  const categoriesId = useId();
  const removeSuggestionsId = useId();
  const addErrorId = useId();
  const removeErrorId = useId();

  const category = categoryMode === "keep" ? undefined : categoryMode === "clear" ? "" : categoryName.trim();
  const addTags = tagsFrom(addText);
  const removeTags = tagsFrom(removeText);
  const conflicts = removeTags.filter((tag) => addTags.includes(tag));
  const addError = tagLimitError(addTags);
  const removeError = tagLimitError(removeTags) || (conflicts.length ? `同一标签不能同时添加和移除：${conflicts.join("、")}` : "");
  const hasChanges = category !== undefined || addTags.length > 0 || removeTags.length > 0;
  const canSave = skills.length > 0 && hasChanges && (categoryMode !== "set" || !!category) && !addError && !removeError;
  const removePrefix = removeText.slice(0, Math.max(removeText.lastIndexOf(","), removeText.lastIndexOf("，")) + 1);
  const alreadyRemoved = new Set(tagsFrom(removePrefix));
  const existingTags = [...new Set(skills.flatMap((skill) => skill.tags ?? []))]
    .filter((tag) => !alreadyRemoved.has(tag) && !/[,，\n]/.test(tag))
    .sort((a, b) => a.localeCompare(b));

  async function save(event: FormEvent<HTMLFormElement>) {
    event.preventDefault();
    if (busy || !canSave) return;
    setBusy(true);
    setError("");
    try {
      await batchUpdateSkillMetadata({ changes: skills.map(({ id }) => ({ skillId: id, category, addTags, removeTags })) });
      onSaved();
    } catch (err) {
      setError(message(err));
    } finally {
      setBusy(false);
    }
  }

  return <form className="github-form batch-metadata" onSubmit={save} onChange={() => setError("")}>
    <details className="skill-disclosure batch-metadata-scope">
      <summary>已选 {skills.length} 项 Skill</summary>
      <ul>{skills.map(({ id, name }) => <li key={id}>{name}</li>)}</ul>
    </details>
    <fieldset disabled={busy} className="skill-fields">
      <label>分类<select value={categoryMode} onChange={(event) => setCategoryMode(event.target.value)}>
        <option value="keep">保持原值</option>
        <option value="set">设为指定分类</option>
        <option value="clear">设为未分类</option>
      </select></label>
      {categoryMode === "set" && <label>分类名称<input value={categoryName}
        onChange={(event) => setCategoryName(event.target.value)} list={categoriesId} maxLength={128} required /></label>}
      <datalist id={categoriesId}>{categories.map((name) => <option key={name} value={name} />)}</datalist>
      <label>添加标签<input value={addText} onChange={(event) => setAddText(event.target.value)} placeholder="用逗号分隔"
        aria-invalid={!!addError} aria-describedby={addError ? addErrorId : undefined} /></label>
      {addError && <p id={addErrorId} className="intake-error" role="alert">{addError}</p>}
      <label>移除标签<input value={removeText} onChange={(event) => setRemoveText(event.target.value)} placeholder="选择或输入标签，用逗号分隔"
        list={removeSuggestionsId} aria-invalid={!!removeError} aria-describedby={removeError ? removeErrorId : undefined} /></label>
      <datalist id={removeSuggestionsId}>{existingTags.map((tag) =>
        <option key={tag} value={removePrefix ? `${removePrefix} ${tag}` : tag} />,
      )}</datalist>
      {removeError && <p id={removeErrorId} className="intake-error" role="alert">{removeError}</p>}
    </fieldset>
    <Failure error={error} />
    <button type="submit" disabled={busy || !canSave}>{busy ? "保存中…" : `应用到 ${skills.length} 项`}</button>
  </form>;
}
