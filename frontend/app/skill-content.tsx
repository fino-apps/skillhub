import { useEffect, useRef, useState } from "react";
import ReactMarkdown from "react-markdown";
import remarkGfm from "remark-gfm";
import { getVersionContent, getVersionFile } from "@/generated/api-client";
import { message } from "./library-api";
import { Failure } from "./skill-ui";

type Preview = { text?: string; size?: number; binary?: boolean; tooLarge?: boolean };
type Contents = { description?: string; files?: string[] };
type Tree = { files: string[]; directories: Map<string, Tree> };

function fileTree(files: string[]) {
  const root: Tree = { files: [], directories: new Map() };
  for (const file of files) {
    let current = root;
    for (const part of file.split("/").slice(0, -1)) {
      let folder = current.directories.get(part);
      if (!folder) {
        folder = { files: [], directories: new Map() };
        current.directories.set(part, folder);
      }
      current = folder;
    }
    current.files.push(file);
  }
  return root;
}

function FileTree({ tree, selected, onSelect }: { tree: Tree; selected: string; onSelect: (path: string) => void }) {
  return <ul className="skill-file-tree">
    {[...tree.directories].map(([name, child]) => <li key={name}>
      <details><summary>{name}/</summary><FileTree tree={child} selected={selected} onSelect={onSelect} /></details>
    </li>)}
    {tree.files.map((path) => <li key={path}>
      <button type="button" className="skillhub-text-button" aria-pressed={selected === path}
        onClick={() => onSelect(path)}>{path.split("/").at(-1)}</button>
    </li>)}
  </ul>;
}

function resolveRelative(from: string, href: string) {
  let relative: string;
  try {
    relative = decodeURIComponent(href.split(/[?#]/, 1)[0]);
  } catch {
    return "";
  }
  if (!relative || relative.startsWith("/") || relative.includes("\\")) return "";
  const parts = from.split("/").slice(0, -1);
  for (const segment of relative.split("/")) {
    if (segment === "..") {
      if (!parts.length) return "";
      parts.pop();
    } else if (segment && segment !== ".") {
      parts.push(segment);
    }
  }
  return parts.join("/");
}

function markdownBody(text: string) {
  return text.replace(/^(?:\uFEFF)?---\r?\n[\s\S]*?\r?\n---(?:\r?\n|$)/, "");
}

export default function SkillContent({ skillId, versionId, available }: { skillId: string; versionId: string; available: boolean }) {
  const [contents, setContents] = useState<Contents>({});
  const [path, setPath] = useState("SKILL.md");
  const [history, setHistory] = useState<{ path: string; y: number }[]>([]);
  const [preview, setPreview] = useState<Preview | null>(null);
  const [error, setError] = useState("");
  const position = useRef<number | null>(null);

  useEffect(() => {
    if (!available) return;
    const controller = new AbortController();
    void getVersionContent<Contents>({ skillId, versionId }, { signal: controller.signal })
      .then(setContents, (err) => { if (!controller.signal.aborted) setError(message(err)); });
    return () => controller.abort();
  }, [skillId, versionId, available]);

  useEffect(() => {
    if (!available) return;
    const controller = new AbortController();
    setPreview(null);
    setError("");
    void getVersionFile<Preview>({ skillId, versionId, path }, { signal: controller.signal }).then((result) => {
      if (!controller.signal.aborted) setPreview(result);
    }, (err) => { if (!controller.signal.aborted) setError(message(err)); });
    return () => controller.abort();
  }, [skillId, versionId, path, available]);

  useEffect(() => {
    if (preview && position.current !== null) {
      window.scrollTo(0, position.current);
      position.current = null;
    }
  }, [preview]);

  function open(file: string) {
    if (file === path) return;
    setHistory((previous) => [...previous, { path, y: window.scrollY }]);
    setPath(file);
  }

  function back() {
    const prior = history.at(-1);
    if (!prior) return;
    position.current = prior.y;
    setHistory((previous) => previous.slice(0, -1));
    setPath(prior.path);
  }

  if (!available) return <p className="skillhub-empty">此版本的快照待补齐。</p>;
  const files = contents.files ?? [];
  const tree = fileTree(files);
  return <section className="skill-content">
    {contents.description && <details className="skill-disclosure"><summary>原始描述</summary><p>{contents.description}</p></details>}
    {files.length > 1 && <details className="skill-disclosure">
      <summary>浏览文件 <span className="skillhub-hint">{files.length}</span></summary>
      <FileTree tree={tree} selected={path} onSelect={open} />
    </details>}
    <Failure error={error} />
    {path !== "SKILL.md" && <div className="library-actions skill-file-heading">
      <button type="button" className="skillhub-text-button" onClick={back}>返回</button><code>{path}</code>
    </div>}
    {!preview && !error && <p role="status">读取中…</p>}
    {preview?.binary && <p>二进制文件 · {preview.size ?? 0} 字节</p>}
    {preview?.tooLarge && <p>文件过大，可保存版本到本机查看。</p>}
    {preview && !preview.binary && !preview.tooLarge && (path.toLowerCase().endsWith(".md")
      ? <article className="skill-markdown">
        <ReactMarkdown skipHtml disallowedElements={["img"]} remarkPlugins={[remarkGfm]} components={{
          a({ href, children }) {
            if (!href) return <span>{children}</span>;
            if (href.startsWith("#")) return <a href={href}>{children}</a>;
            if (/^https?:\/\//i.test(href)) return <a href={href} target="_blank" rel="noreferrer">{children}</a>;
            const file = resolveRelative(path, href);
            return file ? <button type="button" className="skillhub-text-button markdown-file-link" onClick={() => open(file)}>{children}</button> : <span>{children}</span>;
          },
        }}>{path === "SKILL.md" ? markdownBody(preview.text ?? "") : preview.text ?? ""}</ReactMarkdown>
      </article>
      : <pre className="copy-command"><code>{preview.text}</code></pre>)}
  </section>;
}
