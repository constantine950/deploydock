"use client";

import { useEffect, useState } from "react";
import { useParams } from "next/navigation";
import api from "@/lib/api";
import { statusColor, timeAgo, shortSha } from "@/lib/utils";

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

interface Deployment {
  id: string;
  commit_sha: string;
  commit_message: string;
  status: string;
  port: number;
  error_message: string;
  created_at: string;
  finished_at: string;
}

export default function AppDetailPage() {
  const { id } = useParams<{ id: string }>();
  const [app, setApp] = useState<App | null>(null);
  const [deployments, setDeployments] = useState<Deployment[]>([]);
  const [loading, setLoading] = useState(true);
  const [deploying, setDeploying] = useState(false);

  async function load() {
    try {
      const [appRes, depRes] = await Promise.all([
        api.get(`/apps/${id}`),
        api.get(`/apps/${id}/deployments`),
      ]);
      setApp(appRes.data);
      setDeployments(depRes.data.deployments || []);
    } catch (err) {
      console.error(err);
    } finally {
      setLoading(false);
    }
  }

  useEffect(() => {
    load();
  }, [id]);

  async function triggerDeploy() {
    if (!app) return;
    setDeploying(true);
    try {
      await api.post(`/apps/${id}/deploy`);
      setTimeout(load, 2000);
    } catch (err: any) {
      alert(err.response?.data?.error || "Deploy failed");
    } finally {
      setDeploying(false);
    }
  }

  async function rollback(deploymentId: string) {
    if (!confirm("Roll back to this deployment?")) return;
    try {
      await api.post(`/deployments/${deploymentId}/rollback`);
      load();
    } catch (err: any) {
      alert(err.response?.data?.error || "Rollback failed");
    }
  }

  if (loading)
    return <div className="p-8 text-gray-500 text-sm">Loading...</div>;
  if (!app)
    return <div className="p-8 text-gray-500 text-sm">App not found</div>;

  return (
    <div className="p-8">
      <div className="flex items-center justify-between mb-8">
        <div>
          <div className="flex items-center gap-3">
            <h1 className="text-2xl font-bold text-white">{app.name}</h1>
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
        <button
          onClick={triggerDeploy}
          disabled={
            deploying || app.status === "building" || app.status === "deploying"
          }
          className="bg-blue-600 hover:bg-blue-700 disabled:opacity-50 text-white text-sm font-medium px-4 py-2 rounded-lg transition-colors"
        >
          {deploying ? "Queuing..." : "Deploy now"}
        </button>
      </div>

      <div>
        <h2 className="text-sm font-medium text-gray-400 uppercase tracking-wider mb-4">
          Deployments
        </h2>
        {deployments.length === 0 ? (
          <div className="text-center py-16 text-gray-600 text-sm border border-gray-800 rounded-xl">
            No deployments yet — push to your repo or click Deploy now
          </div>
        ) : (
          <div className="space-y-2">
            {deployments.map((d) => (
              <div
                key={d.id}
                className="bg-[#111] border border-gray-800 rounded-xl p-4 flex items-center justify-between"
              >
                <div className="flex items-center gap-4">
                  <span
                    className={`text-xs px-2 py-0.5 rounded-full font-medium ${statusColor(d.status as any)}`}
                  >
                    {d.status}
                  </span>
                  <div>
                    <div className="flex items-center gap-2">
                      <span className="text-white text-sm font-mono">
                        {shortSha(d.commit_sha || "0000000")}
                      </span>
                      <span className="text-gray-400 text-sm">
                        {d.commit_message || "Manual deploy"}
                      </span>
                    </div>
                    <div className="flex items-center gap-3 mt-0.5">
                      <span className="text-xs text-gray-600">
                        {timeAgo(d.created_at)}
                      </span>
                      {d.port && (
                        <span className="text-xs text-gray-600">
                          port {d.port}
                        </span>
                      )}
                      {d.error_message && (
                        <span className="text-xs text-red-400 truncate max-w-xs">
                          {d.error_message}
                        </span>
                      )}
                    </div>
                  </div>
                </div>
                <div className="flex items-center gap-2">
                  <a
                    href={`/dashboard/deployments/${d.id}`}
                    className="text-xs text-gray-500 hover:text-white transition-colors px-3 py-1.5 border border-gray-800 hover:border-gray-700 rounded-lg"
                  >
                    Logs
                  </a>
                  {(d.status === "live" || d.status === "failed") && (
                    <button
                      onClick={() => rollback(d.id)}
                      className="text-xs text-gray-500 hover:text-white transition-colors px-3 py-1.5 border border-gray-800 hover:border-gray-700 rounded-lg"
                    >
                      Rollback
                    </button>
                  )}
                </div>
              </div>
            ))}
          </div>
        )}
      </div>
    </div>
  );
}
