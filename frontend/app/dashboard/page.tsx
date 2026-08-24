"use client";

import { useEffect, useState } from "react";
import Link from "next/link";
import api from "@/lib/api";
import { statusColor, timeAgo } from "@/lib/utils";

interface App {
  id: string;
  name: string;
  slug: string;
  status: string;
  runtime: string;
  repo_url: string;
  branch: string;
  created_at: string;
}

export default function DashboardPage() {
  const [apps, setApps] = useState<App[]>([]);
  const [loading, setLoading] = useState(true);

  useEffect(() => {
    api
      .get("/apps")
      .then((r) => setApps(r.data.apps || []))
      .catch(console.error)
      .finally(() => setLoading(false));
  }, []);

  return (
    <div className="p-8">
      <div className="flex items-center justify-between mb-8">
        <div>
          <h1 className="text-2xl font-bold text-white">Dashboard</h1>
          <p className="text-gray-400 mt-1">All your deployed apps</p>
        </div>
        <Link
          href="/dashboard/apps/new"
          className="bg-blue-600 hover:bg-blue-700 text-white text-sm font-medium px-4 py-2 rounded-lg transition-colors"
        >
          + New App
        </Link>
      </div>

      {loading ? (
        <div className="text-gray-500 text-sm">Loading...</div>
      ) : apps.length === 0 ? (
        <div className="text-center py-24 text-gray-500">
          <p className="text-lg mb-2">No apps yet</p>
          <p className="text-sm">Create your first app to get started</p>
        </div>
      ) : (
        <div className="grid gap-4">
          {apps.map((app) => (
            <Link
              key={app.id}
              href={`/dashboard/apps/${app.id}`}
              className="bg-[#111] border border-gray-800 rounded-xl p-5 hover:border-gray-700 transition-colors"
            >
              <div className="flex items-center justify-between">
                <div>
                  <div className="flex items-center gap-3">
                    <h2 className="text-white font-medium">{app.name}</h2>
                    <span
                      className={`text-xs px-2 py-0.5 rounded-full font-medium ${statusColor(app.status as any)}`}
                    >
                      {app.status}
                    </span>
                    {app.runtime && (
                      <span className="text-xs px-2 py-0.5 rounded-full bg-gray-800 text-gray-400">
                        {app.runtime}
                      </span>
                    )}
                  </div>
                  <p className="text-sm text-gray-500 mt-1">
                    {app.repo_url} · {app.branch}
                  </p>
                </div>
                <span className="text-xs text-gray-600">
                  {timeAgo(app.created_at)}
                </span>
              </div>
            </Link>
          ))}
        </div>
      )}
    </div>
  );
}
