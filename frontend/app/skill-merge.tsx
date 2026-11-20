import { useEffect, useState } from "react";
import { mergeSkill, previewSkillMerge } from "@/generated/api-client";
import { fetchAllSkills, message, sourceLabel, type Skill } from "./library-api";
import { Failure, useSkillOperation } from "./skill-ui";

export type VersionMergeSummary = { addedVersions?: number; restoredSnapshots?: number; skippedVersions?: number };

export function mergeSummaryText(summary: VersionMergeSummary) {
  return `新增 ${summary.addedVersions ?? 0} 个版本，补齐 ${summary.restoredSnapshots ?? 0} 个快照，跳过 ${summary.skippedVersions ?? 0} 个重复版本。`;
}

export default function SkillMerge({ skill, onMerged }: {
  skill: Skill;
  onMerged: (targetId: string, summary: VersionMergeSummary) => void;
}) {
  const operation = useSkillOperation();
  const [targets, setTargets] = useState<Skill[]>([]);
  const [pending, setPending] = useState(true);
  const [loadError, setLoadError] = useState("");
  const [attempt, setAttempt] = useState(0);
  const [targetId, setTargetId] = useState("");
  const [summary, setSummary] = useState<VersionMergeSummary>();

  useEffect(() => {
    const controller = new AbortController();
    setPending(true);
    setLoadError("");
    void fetchAllSkills(controller.signal).then((skills) => {
      if (!controller.signal.aborted) {
        setTargets(skills.filter(({ id }) => id !== skill.id));
        setPending(false);
      }
    }, (error) => {
      if (!controller.signal.aborted) {
        setLoadError(message(error));
        setPending(false);
      }
    });
    return () => controller.abort();
  }, [attempt, skill.id]);

  const name = skill.name.trim().toLowerCase();
  const matchesName = (target: Skill) => target.name.trim().toLowerCase() === name;
  const sameName = targets.filter(matchesName);
  const otherTargets = targets.filter((target) => !matchesName(target));
  const target = targets.find(({ id }) => id === targetId);
  const options = (skills: Skill[]) => skills.map((item) => (
    <option key={item.id} value={item.id}>{item.name} · {item.repositoryUrl || item.localPath || sourceLabel(item)} · {item.id.slice(-8)}</option>
  ));

  if (pending) return <p role="status">正在读取技能库…</p>;
  if (loadError) return <><Failure error={loadError} /><button type="button" onClick={() => setAttempt((value) => value + 1)}>重试</button></>;
  if (!targets.length) return <p>技能库里没有其他 Skill 可以合并。</p>;

  return <form className="github-form" onSubmit={(event) => {
    event.preventDefault();
    if (!target) return;
    const input = { skillId: skill.id, targetSkillId: target.id };
    if (summary) {
      void operation.run("merge", async () => onMerged(target.id, await mergeSkill<VersionMergeSummary>(input)));
    } else {
      void operation.run("preview", async () => setSummary(await previewSkillMerge<VersionMergeSummary>(input)));
    }
  }}>
    <label>保留的主条目
      <select value={targetId} required disabled={!!operation.busy} onChange={(event) => {
        setTargetId(event.target.value);
        setSummary(undefined);
        operation.dismiss();
      }}>
        <option value="">选择要保留的 Skill</option>
        {sameName.length > 0 && <optgroup label="同名 Skill">{options(sameName)}</optgroup>}
        {otherTargets.length > 0 && <optgroup label="其他 Skill">{options(otherTargets)}</optgroup>}
      </select>
    </label>
    {summary && target && <section className="skill-merge-preview">
      <p role="status">{mergeSummaryText(summary)}</p>
      <p>合并后将移除当前条目「{skill.name}」，保留所选主条目的来源、默认版本和整理信息。</p>
    </section>}
    <button type="submit" disabled={!target || !!operation.busy}>
      {operation.busy === "preview" ? "正在预览…" : operation.busy === "merge" ? "正在合并…" : summary ? "确认合并" : "预览合并"}
    </button>
    <Failure error={operation.error} />
  </form>;
}
