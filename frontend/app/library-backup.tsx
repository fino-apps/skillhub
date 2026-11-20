import { useState } from "react";
import { commitLibraryImport, exportLibrary, previewLibraryImport, previewSkillMerge } from "@/generated/api-client";
import { sourceNames } from "./library-api";
import { mergeSummaryText, type VersionMergeSummary } from "./skill-merge";
import { Failure, useSkillOperation } from "./skill-ui";

type MergeEntry = { id: string; name: string; source?: string; versionCount?: number; defaultVersion?: string };
type Candidate = { targetSkillId: string; summary?: VersionMergeSummary };
type ImportItem = { skill: MergeEntry; candidates?: Candidate[]; exactTargetId?: string };
type BackupPreview = { uploadId: string; items?: ImportItem[]; targets?: MergeEntry[] };
type Choice = { action: string; targetSkillId?: string };
type BackupResult = { addedCount?: number; mergedCount?: number; skippedCount?: number; versions?: VersionMergeSummary };

export async function downloadLibraryBackup() {
  const archive = await exportLibrary();
  const url = URL.createObjectURL(archive);
  const link = document.createElement("a");
  link.href = url;
  link.download = `skillhub-backup-${new Date().toISOString().slice(0, 10)}.zip`;
  link.click();
  window.setTimeout(() => URL.revokeObjectURL(url), 1000);
}

function entryDetails(entry: MergeEntry) {
  return [
    sourceNames[entry.source ?? ""] || entry.source || "来源未记录",
    `${entry.versionCount ?? 0} 个版本`,
    entry.defaultVersion ? `默认 ${entry.defaultVersion.slice(0, 8)}` : "未设默认版本",
  ].join(" · ");
}

function defaultChoice(item: ImportItem): Choice {
  if (item.exactTargetId) return { action: "merge", targetSkillId: item.exactTargetId };
  return { action: item.candidates?.length ? "" : "add" };
}

export default function LibraryBackup({ onImported }: { onImported: (result: BackupResult) => void }) {
  const operation = useSkillOperation();
  const [preview, setPreview] = useState<BackupPreview>();
  const [choices, setChoices] = useState<Record<string, Choice>>({});
  const [estimates, setEstimates] = useState<Record<string, VersionMergeSummary>>({});

  if (!preview) return <form className="github-form" onSubmit={(event) => {
    event.preventDefault();
    const file = new FormData(event.currentTarget).get("backup");
    void operation.run("preview", async () => {
      if (!(file instanceof File) || !file.size) throw new Error("请选择非空的技能库备份 ZIP。");
      if (file.size > 256 * 1024 * 1024) throw new Error("技能库备份超过 256 MiB。");
      setPreview(await previewLibraryImport<BackupPreview>({ file }));
    });
  }}>
    <p className="skillhub-hint">请选择 SkillHub 导出的 ZIP，最大 256 MiB。预览后可选择如何处理重复条目。</p>
    <label>技能库备份（ZIP）<input type="file" name="backup" accept=".zip,application/zip" required disabled={!!operation.busy} /></label>
    <button type="submit" disabled={!!operation.busy}>{operation.busy ? "正在读取备份…" : "预览导入"}</button>
    <Failure error={operation.error} />
  </form>;

  const items = preview.items ?? [];
  const targets = preview.targets ?? [];
  const duplicates = items.filter((item) => item.candidates?.length);
  const newItems = items.filter((item) => !item.candidates?.length);
  const selection = (item: ImportItem) => choices[item.skill.id] ?? defaultChoice(item);
  const estimateKey = (skillId: string, targetId: string) => `${skillId}/${targetId}`;
  const estimateFor = (item: ImportItem, targetId: string) =>
    item.candidates?.find(({ targetSkillId }) => targetSkillId === targetId)?.summary ??
    estimates[estimateKey(item.skill.id, targetId)];
  const unresolved = items.filter((item) => {
    const choice = selection(item);
    return !choice.action || (choice.action === "merge" && (!choice.targetSkillId || !estimateFor(item, choice.targetSkillId)));
  }).length;
  const busy = !!operation.busy;

  function setChoice(id: string, choice: Choice) {
    setChoices((current) => ({ ...current, [id]: choice }));
  }

  function estimateMerge(item: ImportItem, targetId: string) {
    if (!targetId || estimateFor(item, targetId) || !preview) return;
    const skillId = item.skill.id;
    void operation.run("estimate", async () => {
      const summary = await previewSkillMerge<VersionMergeSummary>({ skillId, targetSkillId: targetId, backupUploadId: preview.uploadId });
      setEstimates((current) => ({ ...current, [estimateKey(skillId, targetId)]: summary }));
    });
  }

  function itemOptions(item: ImportItem) {
    const { skill, candidates = [] } = item;
    const choice = selection(item);
    const similar = new Set(candidates.map(({ targetSkillId }) => targetSkillId));
    const selected = targets.find(({ id }) => id === choice.targetSkillId);
    const estimate = selected && estimateFor(item, selected.id);
    const options = (entries: MergeEntry[]) => entries.map((entry) => (
      <option key={entry.id} value={entry.id}>{entry.name} · {entryDetails(entry)} · {entry.id.slice(-8)}</option>
    ));

    return <fieldset key={skill.id} className="backup-import-item" disabled={busy}>
      <legend>{skill.name}</legend>
      <p className="skillhub-hint">{entryDetails(skill)}</p>
      <label>处理方式
        <select value={choice.action} onChange={(event) => {
          const action = event.target.value;
          setChoice(skill.id, {
            action,
            targetSkillId: action === "merge" && candidates.length === 1 ? candidates[0].targetSkillId : undefined,
          });
        }}>
          <option value="" disabled>选择处理方式</option>
          <option value="add">作为独立条目导入</option>
          {targets.length > 0 && <option value="merge">合并到已有 Skill</option>}
          <option value="skip">跳过</option>
        </select>
      </label>
      {choice.action === "merge" && <label>保留的主条目
        <select value={choice.targetSkillId ?? ""} onChange={(event) => {
          const targetId = event.target.value;
          setChoice(skill.id, { action: "merge", targetSkillId: targetId });
          estimateMerge(item, targetId);
        }}>
          <option value="">选择要保留的 Skill</option>
          {candidates.length > 0 && <optgroup label="可能重复">{options(targets.filter(({ id }) => similar.has(id)))}</optgroup>}
          {similar.size < targets.length && <optgroup label="其他 Skill">{options(targets.filter(({ id }) => !similar.has(id)))}</optgroup>}
        </select>
      </label>}
      {selected && <p className="skillhub-hint">{entryDetails(selected)}</p>}
      {estimate && <p className="skillhub-hint">预计{mergeSummaryText(estimate)}</p>}
      {selected && !estimate && <button type="button" className="secondary" onClick={() => estimateMerge(item, selected.id)}>
        {busy ? "正在计算…" : "预览版本变动"}
      </button>}
    </fieldset>;
  }

  return <form className="github-form backup-import-form" onSubmit={(event) => {
    event.preventDefault();
    if (unresolved || !items.length) return;
    const decisions = items.flatMap((item) => {
      if (!item.candidates?.length && !choices[item.skill.id]) return [];
      return [{ skillId: item.skill.id, ...selection(item) }];
    });
    void operation.run("import", async () => {
      onImported(await commitLibraryImport<BackupResult>({ uploadId: preview.uploadId, choices: decisions }));
    });
  }}>
    {items.length === 0 && <p>备份中没有 Skill。</p>}
    {duplicates.length > 0 && <section className="backup-import-list" aria-label="可能重复的 Skill">
      <h3>可能重复（{duplicates.length}）</h3>
      <p className="skillhub-hint">合并会补充版本历史，保留主条目的来源、默认版本和整理信息。</p>
      {duplicates.map(itemOptions)}
    </section>}
    {newItems.length > 0 && <details className="backup-import-list">
      <summary>其他条目（{newItems.length}）</summary>
      <p className="skillhub-hint">默认作为独立条目导入，可展开调整。</p>
      {newItems.map(itemOptions)}
    </details>}
    {unresolved > 0 && <p role="status">还有 {unresolved} 项需要完成选择或预览。</p>}
    <Failure error={operation.error} />
    <section className="library-actions">
      <button type="submit" disabled={busy || !!unresolved || items.length === 0}>{busy ? "正在导入…" : "确认导入"}</button>
      <button type="button" className="secondary" disabled={busy} onClick={() => {
        setPreview(undefined);
        setChoices({});
        setEstimates({});
        operation.dismiss();
      }}>更换备份</button>
    </section>
  </form>;
}
