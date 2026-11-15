import { useState, type FormEvent } from "react";
import { getInstallCommands, scanProject } from "@/generated/api-client";

type Props = { skillId: string; skillName: string; versionId: string; version: string };
type ScanResult = { skills?: { name: string; path: string }[] };

export default function InstallPanel({ skillId, skillName, versionId, version }: Props) {
  const [scope, setScope] = useState<"project" | "global">("project");
  const [projectPath, setProjectPath] = useState("");
  const [pathConfirmed, setPathConfirmed] = useState(false);
  const [command, setCommand] = useState("");
  const [warning, setWarning] = useState("");
  const [warningConfirmed, setWarningConfirmed] = useState(false);
  const [error, setError] = useState("");
  const [status, setStatus] = useState("");
  const [busy, setBusy] = useState(false);

  const root = projectPath.trim().replace(/\/+$/, "") || (projectPath.trim() === "/" ? "/" : "");
  const suspiciousPath = /(?:^|\/)skills$/i.test(root);
  const target = root ? `${root === "/" ? "" : root}/.agents/skills/${skillName}` : "";

  function resetResult() {
    setCommand("");
    setWarning("");
    setWarningConfirmed(false);
    setError("");
    setStatus("");
  }

  function changeScope(value: "project" | "global") {
    setScope(value);
    setPathConfirmed(false);
    resetResult();
  }

  async function generate(event: FormEvent<HTMLFormElement>) {
    event.preventDefault();
    if (busy || (scope === "project" && suspiciousPath && !pathConfirmed)) return;
    setBusy(true);
    resetResult();
    try {
      const result = await getInstallCommands<{ commands?: string[] }>({ skills: [{ skillId, versionId }], projectPath: scope === "project" ? root : "", global: scope === "global" });
      if (!result.commands?.[0]) throw new Error("未返回安装命令");
      if (scope === "project") {
        try {
          const scan = await scanProject<ScanResult>({ projectPath: root });
          if (scan.skills?.some((skill) => skill.name === skillName || skill.path.split("/").pop() === skillName)) {
            setWarning("项目中发现同名 Skill，安装可能覆盖现有内容。请确认后继续。");
          }
        } catch {
          setWarning("无法检查项目中的现有 Skill，请先确认目标位置不会覆盖需要保留的内容。");
        }
      } else {
        setWarning("请确认全局安装位置没有需要保留的同名 Skill，安装可能覆盖现有内容。");
      }
      setCommand(result.commands[0]);
    } catch (reason) {
      const detail = reason instanceof Error ? reason.message : "请求失败";
      setError(detail === "bad request" ? "无法生成命令，请检查项目根目录和 Skill 版本。" : detail);
    } finally { setBusy(false); }
  }

  async function copy() {
    try {
      await navigator.clipboard.writeText(command);
      setStatus("已复制，可在终端中运行。");
    } catch {
      setStatus("复制失败，请手动选择命令。");
    }
  }

  return <div className="install-panel">
    <p className="install-skill"><strong>{skillName}</strong><span>{version}</span></p>
    <form onSubmit={generate}>
      <fieldset className="install-scope" disabled={busy}>
        <legend>安装位置</legend>
        <label><input type="radio" name="install-scope" checked={scope === "project"} onChange={() => changeScope("project")} />项目</label>
        <label><input type="radio" name="install-scope" checked={scope === "global"} onChange={() => changeScope("global")} />全局</label>
      </fieldset>
      {scope === "project" && <>
        <label className="install-path">项目根目录<input value={projectPath} disabled={busy} onChange={(event) => { setProjectPath(event.target.value); setPathConfirmed(false); resetResult(); }} placeholder="/path/to/project" required /></label>
        <p className="skillhub-hint">填写项目根目录，不要填写 skills 或 .agents/skills 目录。</p>
        {target && <p className="skillhub-hint install-target">预计安装位置：<code>{target}</code></p>}
        {suspiciousPath && <label className="install-confirm"><input type="checkbox" checked={pathConfirmed} disabled={busy} onChange={(event) => { setPathConfirmed(event.target.checked); resetResult(); }} />路径以 skills 结尾，请确认这是项目根目录。</label>}
      </>}
      <button type="submit" disabled={busy || (scope === "project" && suspiciousPath && !pathConfirmed)}>{busy ? "生成中…" : "生成安装命令"}</button>
    </form>
    {error && <p className="intake-error" role="alert">{error}</p>}
    {warning && <label className="install-confirm"><input type="checkbox" checked={warningConfirmed} onChange={(event) => setWarningConfirmed(event.target.checked)} />{warning}</label>}
    {command && scope === "project" && <p className="skillhub-hint">扫描仅识别有效的 Skill；执行前请再次检查目标目录。</p>}
    {command && (!warning || warningConfirmed) && <div className="install-command"><code>{command}</code><button className="secondary" type="button" onClick={() => void copy()}>复制</button></div>}
    <p className="skillhub-hint">复制命令后在终端运行，需要 Node.js 22.20 或更高版本。</p>
    <p className="skillhub-hint" role="status">{status}</p>
  </div>;
}
