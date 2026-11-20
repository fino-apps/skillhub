import { useEffect, useId, useRef, useState, type ReactNode } from "react";
import { exportVersionToDirectory } from "@/generated/api-client";
import { backendDirectoryLabel, canPickDirectory, message } from "./library-api";

export function Dialog({ title, children, onClose, compact, className = "" }: {
  title: string; children: ReactNode; onClose: () => void; compact?: boolean; className?: string;
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
  return <dialog ref={ref} className={`skillhub-dialog${compact ? " skillhub-dialog-compact" : ""} ${className}`} onClose={onClose} aria-labelledby={titleId}>
    <div className="dialog-header">
      <h2 id={titleId}>{title}</h2>
      <button className="dialog-close" type="button" onClick={() => ref.current?.close()} aria-label="关闭">×</button>
    </div>
    <div className="dialog-body">{children}</div>
  </dialog>;
}

export function Popover({ summary, label, className, children, closeOnSelect = false }: {
  summary: ReactNode; label?: string; className: string; children: ReactNode; closeOnSelect?: boolean;
}) {
  const details = useRef<HTMLDetailsElement>(null);
  const trigger = useRef<HTMLElement>(null);
  const [open, setOpen] = useState(false);

  useEffect(() => {
    if (!open) return;
    function onPointerDown(event: PointerEvent) {
      if (details.current && event.target instanceof Node && !details.current.contains(event.target)) {
        details.current.open = false;
      }
    }
    document.addEventListener("pointerdown", onPointerDown);
    return () => document.removeEventListener("pointerdown", onPointerDown);
  }, [open]);

  function close() {
    if (!details.current?.open) return;
    trigger.current?.focus();
    details.current.open = false;
  }

  return <details ref={details} className={className}
    onToggle={(event) => setOpen(event.currentTarget.open)}
    onBlur={(event) => {
      if (event.relatedTarget && !event.currentTarget.contains(event.relatedTarget)) event.currentTarget.open = false;
    }}
    onKeyDown={(event) => {
      if (event.key === "Escape" && details.current?.open) {
        event.preventDefault();
        close();
      }
    }}
    onClick={(event) => {
      if (closeOnSelect && event.target instanceof Element && event.target.closest("button, a")) close();
    }}>
    <summary ref={trigger} aria-label={label} title={label}>{summary}</summary>
    {children}
  </details>;
}

export function Menu({ label, children }: { label: string; children: ReactNode }) {
  return <Popover className="skill-menu" summary="⋯" label={label} closeOnSelect>
    <div className="skill-menu-items">{children}</div>
  </Popover>;
}

export function Failure({ error }: { error: string }) {
  return error ? <p role="alert" className="intake-error">{error}</p> : null;
}

export function useSkillOperation() {
  const [busy, setBusy] = useState("");
  const [error, setError] = useState("");
  const [notice, setNotice] = useState("");
  const [savedPath, setSavedPath] = useState("");

  function dismiss() {
    setError("");
    setNotice("");
    setSavedPath("");
  }

  async function run(key: string, action: () => Promise<void>) {
    setBusy(key);
    dismiss();
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

  return { busy, error, notice, savedPath, setNotice, run, save, dismiss };
}

export function OperationFeedback({ error, notice, savedPath, floating, dismiss }: {
  error: string; notice: string; savedPath: string; floating?: boolean; dismiss: () => void;
}) {
  const [copyStatus, setCopyStatus] = useState("");
  useEffect(() => setCopyStatus(""), [savedPath]);
  const visible = error || notice || savedPath;

  async function copyPath() {
    try {
      await navigator.clipboard.writeText(savedPath);
      setCopyStatus("已复制");
    } catch {
      setCopyStatus("无法复制，请选择路径后手动复制");
    }
  }

  return <div className={floating ? "operation-feedback operation-feedback-floating" : "operation-feedback"} hidden={!visible}>
    <div className="operation-feedback-content">
      <Failure error={error} />
      {notice && <p role="status">{notice}</p>}
      {savedPath && <p role="status">已导出到 <code>{savedPath}</code></p>}
      <span role="status" className={!copyStatus || copyStatus === "已复制" ? "sr-only" : "skillhub-hint"}>{copyStatus}</span>
    </div>
    {savedPath && <button type="button" className="skillhub-text-button" onClick={() => void copyPath()}>{copyStatus === "已复制" ? "已复制" : "复制路径"}</button>}
    {floating && <button type="button" className="dialog-close" aria-label="关闭操作提示" onClick={dismiss}>×</button>}
  </div>;
}

export function ExportLocation({ onSave, busy }: { onSave: (directory?: string) => void; busy: boolean }) {
  return <form className="github-form" onSubmit={(event) => {
    event.preventDefault();
    const values = new FormData(event.currentTarget);
    onSave(String(values.get("directory")).trim());
  }}>
    {canPickDirectory && <button type="button" className="secondary" disabled={busy} onClick={() => onSave()}>选择文件夹</button>}
    <label>{backendDirectoryLabel}<input name="directory" disabled={busy} required /></label>
    <button type="submit" disabled={busy}>{busy ? "导出中…" : "导出"}</button>
  </form>;
}
