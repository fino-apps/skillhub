import { useEffect, useId, useRef, useState, type ReactNode } from "react";
import { exportVersionToDirectory } from "@/generated/api-client";
import { message } from "./library-api";

export function Dialog({ title, children, onClose, compact }: {
  title: string; children: ReactNode; onClose: () => void; compact?: boolean;
}) {
  const ref = useRef<HTMLDialogElement>(null);
  const titleId = useId();
  useEffect(() => {
    const dialog = ref.current;
    const trigger = document.activeElement;
    if (!dialog?.open) dialog?.showModal();
    return () => {
      if (trigger instanceof HTMLElement && trigger.isConnected) trigger.focus();
    };
  }, []);
  return <dialog ref={ref} className={`skillhub-dialog${compact ? " skillhub-dialog-compact" : ""}`} onClose={onClose} aria-labelledby={titleId}>
    <div className="dialog-header">
      <h2 id={titleId}>{title}</h2>
      <button className="dialog-close" type="button" onClick={() => ref.current?.close()} aria-label="关闭">×</button>
    </div>
    <div className="dialog-body">{children}</div>
  </dialog>;
}

export function Menu({ label, children }: { label: string; children: ReactNode }) {
  const details = useRef<HTMLDetailsElement>(null);
  return <details ref={details} className="skill-menu"
    onBlur={(event) => {
      if (event.relatedTarget && !event.currentTarget.contains(event.relatedTarget)) event.currentTarget.open = false;
    }}
    onKeyDown={(event) => {
      if (event.key === "Escape" && details.current?.open) {
        details.current.open = false;
        details.current.querySelector("summary")?.focus();
      }
    }}>
    <summary aria-label={label} title={label}>⋯</summary>
    <div className="skill-menu-items" onClick={(event) => {
      if ((event.target as HTMLElement).closest("button, a") && details.current) details.current.open = false;
    }}>{children}</div>
  </details>;
}

export function Failure({ error }: { error: string }) {
  return error ? <p role="alert" className="intake-error">{error}</p> : null;
}

export function useSkillOperation() {
  const [busy, setBusy] = useState("");
  const [error, setError] = useState("");
  const [notice, setNotice] = useState("");
  const [savedPath, setSavedPath] = useState("");

  async function run(key: string, action: () => Promise<void>) {
    setBusy(key);
    setError("");
    setNotice("");
    setSavedPath("");
    try {
      await action();
    } catch (err) {
      setError(message(err));
    } finally {
      setBusy("");
    }
  }

  function save(skillId: string, versionId: string, chooseDirectory = false, directory?: string) {
    return run(versionId, async () => {
      const result = await exportVersionToDirectory<{ path?: string }>({ skillId, versionId, chooseDirectory, directory });
      setSavedPath(result.path ?? "");
    });
  }

  return { busy, error, notice, savedPath, setNotice, run, save };
}

export function OperationFeedback({ error, notice, savedPath }: { error: string; notice: string; savedPath: string }) {
  const [copied, setCopied] = useState(false);
  return <>
    <Failure error={error} />
    {notice && <p className="skill-feedback" role="status">{notice}</p>}
    {savedPath && <div className="skill-feedback" role="status">
      <span>已保存到 <code>{savedPath}</code></span>
      <button type="button" className="skillhub-text-button" onClick={() => {
        void navigator.clipboard.writeText(savedPath).then(() => setCopied(true), () => setCopied(false));
      }}>{copied ? "已复制" : "复制路径"}</button>
    </div>}
  </>;
}

export function ExportLocation({ onSave, busy }: { onSave: (directory?: string) => void; busy: boolean }) {
  return <form className="github-form" onSubmit={(event) => {
    event.preventDefault();
    const values = new FormData(event.currentTarget);
    onSave(String(values.get("directory")).trim());
  }}>
    <button type="button" className="secondary" disabled={busy} onClick={() => onSave()}>选择文件夹</button>
    <label>或输入本机目录<input name="directory" disabled={busy} required /></label>
    <button type="submit" disabled={busy}>{busy ? "保存中…" : "保存到指定目录"}</button>
  </form>;
}
