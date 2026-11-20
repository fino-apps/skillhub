import { exportLibrary, importLibrary } from "@/generated/api-client";
import { OperationFeedback, useSkillOperation } from "./skill-ui";

type BackupResult = { addedCount?: number; existingCount?: number };

export default function LibraryBackup({ onImported }: { onImported: (result: BackupResult) => void }) {
  const operation = useSkillOperation();

  function exportBackup() {
    void operation.run("export", async () => {
      const archive = await exportLibrary();
      const url = URL.createObjectURL(archive);
      const link = document.createElement("a");
      link.href = url;
      link.download = `skillhub-backup-${new Date().toISOString().slice(0, 10)}.zip`;
      link.click();
      window.setTimeout(() => URL.revokeObjectURL(url), 1000);
      operation.setNotice("已发起备份下载。");
    });
  }

  return <div className="github-form">
    <p className="skillhub-hint">备份包含整个技能库的元数据、历史版本和已保存的快照，最大 256 MiB。恢复时跳过已有的 Skill。</p>
    <button type="button" className="secondary" disabled={!!operation.busy} onClick={exportBackup}>
      {operation.busy === "export" ? "正在导出…" : "导出全部数据"}
    </button>
    <form className="github-form" onSubmit={(event) => {
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
      <label>技能库备份（ZIP）<input type="file" name="backup" accept=".zip,application/zip" required disabled={!!operation.busy} /></label>
      <button type="submit" disabled={!!operation.busy}>{operation.busy === "import" ? "正在恢复…" : "导入备份"}</button>
    </form>
    {(operation.error || operation.notice) && <OperationFeedback {...operation} />}
  </div>;
}
