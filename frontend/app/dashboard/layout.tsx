"use client";

import { useEffect, useState } from "react";
import { useRouter, usePathname } from "next/navigation";
import Link from "next/link";
import { getToken, logout } from "@/lib/auth";
import { LayoutDashboard, Rocket, LogOut, ChevronRight } from "lucide-react";

export default function DashboardLayout({
  children,
}: {
  children: React.ReactNode;
}) {
  const router = useRouter();
  const pathname = usePathname();
  const [ready, setReady] = useState(false);

  useEffect(() => {
    if (!getToken()) {
      router.push("/login");
    } else {
      setReady(true);
    }
  }, [router]);

  if (!ready) return null;

  return (
    <div className="min-h-screen bg-[#0a0a0a] flex">
      <aside className="w-64 border-r border-gray-800 flex flex-col">
        <div className="h-16 flex items-center px-6 border-b border-gray-800">
          <span className="text-white font-bold text-lg tracking-tight">
            🚀 DeployDock
          </span>
        </div>
        <nav className="flex-1 p-4 space-y-1">
          <NavItem
            href="/dashboard"
            icon={<LayoutDashboard size={16} />}
            label="Dashboard"
            active={pathname === "/dashboard"}
          />
          <NavItem
            href="/dashboard/apps"
            icon={<Rocket size={16} />}
            label="Apps"
            active={pathname.startsWith("/dashboard/apps")}
          />
        </nav>
        <div className="p-4 border-t border-gray-800">
          <button
            onClick={logout}
            className="flex items-center gap-2 text-sm text-gray-400 hover:text-white transition-colors w-full px-3 py-2 rounded-lg hover:bg-gray-800"
          >
            <LogOut size={16} />
            Sign out
          </button>
        </div>
      </aside>
      <main className="flex-1 overflow-auto">{children}</main>
    </div>
  );
}

function NavItem({
  href,
  icon,
  label,
  active,
}: {
  href: string;
  icon: React.ReactNode;
  label: string;
  active: boolean;
}) {
  return (
    <Link
      href={href}
      className={`flex items-center gap-2 px-3 py-2 rounded-lg text-sm transition-colors ${active ? "bg-blue-600/10 text-blue-400 border border-blue-600/20" : "text-gray-400 hover:text-white hover:bg-gray-800"}`}
    >
      {icon}
      <span className="flex-1">{label}</span>
      {active && <ChevronRight size={14} />}
    </Link>
  );
}
