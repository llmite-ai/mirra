"use client";

import React from "react";
import { Routes, Route, Link, NavLink, useLocation } from "react-router";
import Recordings from "./pages/Recordings";
import Recording from "./pages/Recording";
import Sessions from "./pages/Sessions";
import Session from "./pages/Session";
import { ModeToggle } from "./components/mode-toggle";
import { Logo } from "./components/Logo";
import { useSessionsEnabled } from "./lib/useSessionsEnabled";
import { cn } from "./lib/utils";

function App() {
  const sessionsEnabled = useSessionsEnabled();

  return (
    <div className="min-h-screen flex flex-col bg-background">
      {/* Header */}
      <header className="border-b">
        <div className=" mx-auto px-4">
          <div className="flex h-16 items-center justify-between">
            <div className="flex items-center gap-8">
              <Link to="/" aria-label="mirra home">
                <Logo />
              </Link>
              <nav className="flex items-center gap-1">
                <HeaderLink to="/recordings">Recordings</HeaderLink>
                {sessionsEnabled && (
                  <HeaderLink to="/sessions">Sessions</HeaderLink>
                )}
              </nav>
            </div>
            <div className="flex items-center gap-4">
              <ModeToggle />
            </div>
          </div>
        </div>
      </header>

      {/* Main Content */}
      <main>
        <Routes>
          <Route path="/" element={<Recordings />} />
          <Route path="/recordings" element={<Recordings />} />
          <Route path="/recordings/:id" element={<Recording />} />
          <Route path="/sessions" element={<Sessions />} />
          <Route path="/sessions/:traceId" element={<Session />} />
        </Routes>
      </main>
    </div>
  );
}

function HeaderLink({
  to,
  children,
}: {
  to: string;
  children: React.ReactNode;
}) {
  const { pathname } = useLocation();
  // The recordings page also serves "/"
  const isActive =
    pathname.startsWith(to) || (to === "/recordings" && pathname === "/");
  return (
    <NavLink
      to={to}
      className={cn(
        "px-3 py-1.5 text-sm font-medium transition-colors",
        isActive
          ? "text-foreground bg-muted"
          : "text-muted-foreground hover:text-foreground",
      )}
    >
      {children}
    </NavLink>
  );
}

export default App;
