import { exportLibrary, importLibrary } from "@/generated/api-client";
import { Failure, useSkillOperation } from "./skill-ui";

type BackupResult = { addedCount?: number; existingCount?: number };

export async function downloadLibraryBackup() {
  const archive = await exportLibrary();
  const url = URL.createObjectURL(archive);
  const link = document.createElement("a");
  link.href = url;
  link.download = `skillhub-backup-${new Date().toISOString().slice(0, 10)}.zip`;
  link.click();
  window.setTimeout(() => URL.revokeObjectURL(url), 1000);
}

export default function LibraryBackup({ onImported }: { onImported: (result: BackupResult) => void }) {
  const operation = useSkillOperation();

  return <form className="github-form" onSubmit={(event) => {
    event.preventDefault();
    const form = event.currentTarget;
    const file = new FormData(form).get("backup");
    void operation.run("import", async () => {
      if (!(file instanceof File) || !file.size) throw new Error("请选择非空的技能库备份 ZIP。");
      if (file.size > 256 * 1024 * 1024) throw new Error("技能库备份超过 256 MiB。");
      const result = await importLibrary<BackupResult>({ file });
      form.reset();
      onImported(result);
    });
  }}>
    <p className="skillhub-hint">请选择 SkillHub 导出的 ZIP，最大 256 MiB。恢复时跳过已有的 Skill。</p>
    <label>技能库备份（ZIP）<input type="file" name="backup" accept=".zip,application/zip" required disabled={!!operation.busy} /></label>
    <button type="submit" disabled={!!operation.busy}>{operation.busy === "import" ? "正在恢复…" : "导入备份"}</button>
    <Failure error={operation.error} />
  </form>;
}
