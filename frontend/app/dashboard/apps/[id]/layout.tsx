"use client";

import { useParams, usePathname } from "next/navigation";
import Link from "next/link";

export default function AppLayout({ children }: { children: React.ReactNode }) {
  const { id } = useParams<{ id: string }>();
  const pathname = usePathname();

  const tabs = [
    { label: "Deployments", href: `/dashboard/apps/${id}` },
    { label: "Environment", href: `/dashboard/apps/${id}/env` },
  ];

  return (
    <div className="flex flex-col h-full">
      <div className="border-b border-gray-800 px-8 flex gap-1">
        {tabs.map((tab) => {
          const active =
            tab.href === `/dashboard/apps/${id}`
              ? pathname === tab.href
              : pathname.startsWith(tab.href);
          return (
            <Link
              key={tab.href}
              href={tab.href}
              className={`px-4 py-3 text-sm font-medium border-b-2 transition-colors ${active ? "border-blue-500 text-white" : "border-transparent text-gray-500 hover:text-gray-300"}`}
            >
              {tab.label}
            </Link>
          );
        })}
      </div>
      <div className="flex-1 overflow-auto">{children}</div>
    </div>
  );
}
