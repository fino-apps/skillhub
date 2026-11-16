import { useState, type FormEvent } from "react";
import { updateSkill } from "@/generated/api-client";

type Skill = { name: string; description: string; repositoryUrl?: string; localPath?: string; repositoryPath?: string; id: string; category?: string; notes?: string; tags?: string[] };

export default function SkillMetadata({ skill, onSaved }: { skill: Skill; onSaved: (skill: Skill) => void }) {
  const [category, setCategory] = useState(skill.category ?? "");
  const [notes, setNotes] = useState(skill.notes ?? "");
  const [tags, setTags] = useState(skill.tags?.join("\n") ?? "");
  const [saving, setSaving] = useState(false);
  const [error, setError] = useState("");

  async function save(event: FormEvent<HTMLFormElement>) {
    event.preventDefault();
    setSaving(true);
    setError("");
    try {
      const result = await updateSkill<{ skill: Skill }>({
        skillId: skill.id,
        skill: { category, notes, tags: tags.split("\n").map((value) => value.trim()).filter(Boolean) },
      });
      onSaved(result.skill);
    } catch {
      setError("无法保存。请确认标签最多 100 个，每个不超过 128 个字符。");
    } finally {
      setSaving(false);
    }
  }

  return <form className="github-form" onSubmit={save}>
    <label>分类<input value={category} maxLength={128} disabled={saving} onChange={(event) => setCategory(event.target.value)} /></label>
    <label>标签（每行一个）<textarea value={tags} rows={3} disabled={saving} onChange={(event) => setTags(event.target.value)} /></label>
    <label>备注<textarea value={notes} placeholder="可记录请求示例或使用说明" maxLength={4096} rows={3} disabled={saving} onChange={(event) => setNotes(event.target.value)} /></label>
    {error && <p className="intake-error" role="alert">{error}</p>}
    <button type="submit" disabled={saving}>{saving ? "保存中…" : "保存"}</button>
  </form>;
}
