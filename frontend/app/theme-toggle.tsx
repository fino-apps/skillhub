import { useEffect, useState } from "react";

type Theme = "light" | "dark";

function setFavicon(theme: Theme) {
  const favicon = document.querySelector<HTMLLinkElement>("#fino-favicon");
  if (favicon) {
    favicon.href = theme === "dark" ? "/fino-mark-dark.png?v=4" : "/fino-mark.png?v=4";
  }
}

function preferredTheme(): Theme {
  try {
    const savedTheme = window.localStorage.getItem("fino-theme");
    if (savedTheme === "light" || savedTheme === "dark") {
      return savedTheme;
    }
  } catch {
    // Storage can be unavailable in private or restricted browsing contexts.
  }
  return window.matchMedia("(prefers-color-scheme: dark)").matches ? "dark" : "light";
}

export function ThemeToggle({ followSystem = false }: { followSystem?: boolean }) {
  const [theme, setTheme] = useState<Theme>("light");

  useEffect(() => {
    const media = window.matchMedia("(prefers-color-scheme: dark)");
    const apply = () => {
      const nextTheme = followSystem ? (media.matches ? "dark" : "light") : preferredTheme();
      document.documentElement.dataset.theme = nextTheme;
      setFavicon(nextTheme);
      setTheme(nextTheme);
    };
    apply();
    if (followSystem) {
      media.addEventListener("change", apply);
      return () => media.removeEventListener("change", apply);
    }
  }, [followSystem]);

  function toggleTheme() {
    const nextTheme = theme === "dark" ? "light" : "dark";
    document.documentElement.dataset.theme = nextTheme;
    setFavicon(nextTheme);
    try {
      window.localStorage.setItem("fino-theme", nextTheme);
    } catch {
      // Keep the current choice for this page even when persistence is unavailable.
    }
    setTheme(nextTheme);
  }

  if (followSystem) return null;

  return (
    <button className="theme-toggle" type="button" onClick={toggleTheme} aria-pressed={theme === "dark"}>
      <span aria-hidden="true">{theme === "dark" ? "☀" : "☾"}</span>
      <span>{theme === "dark" ? "Light" : "Dark"}</span>
    </button>
  );
}
