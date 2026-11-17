import { useEffect } from "react";
import { Link, NavLink, Outlet, useLocation } from "react-router-dom";
import { ThemeToggle } from "@/app/theme-toggle";

const projectTitle = "skillhub | fino";
const projectName = "skillhub";

export default function AppLayout() {
  const { pathname } = useLocation();

  useEffect(() => {
    const pageTitle = pathname === "/issues" ? "Issues" : pathname === "/console" ? "API Console" : "";
    document.title = pageTitle ? `${pageTitle} | ${projectTitle}` : projectTitle;
  }, [pathname]);

  return (
    <>
      <header className="site-header">
        <div className="site-header__inner">
          <div className="site-branding">
            <Link className="brand" to="/" aria-label="Fino home">
              <img className="fino-logo" src="/fino-logo.png" alt="fino" />
            </Link>
            <Link className="product-label" to="/" title="Home">{projectName}</Link>
          </div>
          <nav className="site-tools" aria-label="开发工具">
            <NavLink className="console-link" to="/console">API Console</NavLink>
            <NavLink className="console-link" to="/issues">Issues</NavLink>
          </nav>
          <nav className="site-nav" aria-label="Main navigation">
            <ThemeToggle />
          </nav>
        </div>
      </header>
      <Outlet />
    </>
  );
}
