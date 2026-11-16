import { useState } from "react";
import { getInstallCommand } from "@/generated/api-client";

export default function InstallCommand({ skillId, versionId }: { skillId: string; versionId: string }) {
  const [command, setCommand] = useState("");
  const [generating, setGenerating] = useState(false);
  const [error, setError] = useState("");
  const [copyStatus, setCopyStatus] = useState("");

  async function generate(global: boolean) {
    setGenerating(true);
    setCommand("");
    setError("");
    setCopyStatus("");
    try {
      const result = await getInstallCommand<{ command: string }>({ skillId, versionId, global });
      setCommand(result.command);
    } catch (reason) {
      setError(reason instanceof Error && reason.message ? reason.message : "请稍后重试");
    } finally {
      setGenerating(false);
    }
  }

  async function copy() {
    try {
      await navigator.clipboard.writeText(command);
      setCopyStatus("已复制");
    } catch {
      setCopyStatus("复制失败，请手动选择命令");
    }
  }

  return <>
    <div className="library-actions" role="group" aria-label="安装范围">
      <button type="button" disabled={generating} onClick={() => void generate(false)}>安装到项目</button>
      <button type="button" className="secondary" disabled={generating} onClick={() => void generate(true)}>全局安装</button>
    </div>
    {generating && <p role="status" className="skillhub-hint">正在生成命令…</p>}
    {error && <p role="alert" className="intake-error">无法生成命令：{error}</p>}
    {command && <>
      <p className="skillhub-hint">在本机终端运行。项目级安装请先进入目标项目，重装可能替换同名 Skill。</p>
      <pre className="copy-command" tabIndex={0}><code>{command}</code></pre>
      <div className="library-actions">
        <button type="button" className="secondary" onClick={() => void copy()}>复制命令</button>
        <span role="status" className="skillhub-hint">{copyStatus}</span>
      </div>
    </>}
  </>;
}
