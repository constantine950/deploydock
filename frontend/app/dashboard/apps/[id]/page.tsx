"use client";

import { useState } from "react";
import { useParams } from "next/navigation";
import api from "@/lib/api";
import { statusColor, timeAgo, shortSha } from "@/lib/utils";
import { usePolling } from "@/hooks/usePolling";
import { Trash2 } from "lucide-react";

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

const ACTIVE_STATUSES = ["building", "deploying", "queued"];

export default function AppDetailPage() {
  const { id } = useParams<{ id: string }>();
  const [app, setApp] = useState<App | null>(null);
  const [deployments, setDeployments] = useState<Deployment[]>([]);
  const [loading, setLoading] = useState(true);
  const [deploying, setDeploying] = useState(false);
  const [deleting, setDeleting] = useState(false);

  const isActive = app ? ACTIVE_STATUSES.includes(app.status) : false;

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

  usePolling(load, isActive ? 3000 : 30000);

  async function triggerDeploy() {
    if (!app) return;
    setDeploying(true);
    try {
      const { data } = await api.post(`/apps/${id}/deploy`);
      window.location.href = `/dashboard/deployments/${data.deployment_id}`;
    } catch (err: any) {
      alert(err.response?.data?.error || "Deploy failed");
      setDeploying(false);
    }
  }

  async function handleDelete() {
    if (!app) return;
    if (!confirm(`Delete "${app.name}"? This cannot be undone.`)) return;
    setDeleting(true);
    try {
      await api.delete(`/apps/${id}`);
      window.location.href = "/dashboard";
    } catch (err: any) {
      alert(err.response?.data?.error || "Delete failed");
      setDeleting(false);
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
            {isActive && (
              <span className="text-xs text-blue-400 animate-pulse">
                deploying...
              </span>
            )}
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
        <div className="flex items-center gap-2">
          <button
            onClick={triggerDeploy}
            disabled={deploying || isActive}
            className="bg-blue-600 hover:bg-blue-700 disabled:opacity-50 text-white text-sm font-medium px-4 py-2 rounded-lg transition-colors"
          >
            {deploying
              ? "Queuing..."
              : isActive
                ? "Deploying..."
                : "Deploy now"}
          </button>
          <button
            onClick={handleDelete}
            disabled={deleting}
            className="flex items-center gap-1.5 border border-red-800 hover:border-red-600 text-red-500 hover:text-red-400 disabled:opacity-50 text-sm font-medium px-3 py-2 rounded-lg transition-colors"
          >
            <Trash2 size={14} />
            {deleting ? "Deleting..." : "Delete"}
          </button>
        </div>
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
                  {d.status === "live" && d.port && (
                    <a
                      href={`http://localhost:${d.port}`}
                      target="_blank"
                      rel="noopener noreferrer"
                      className="text-xs text-green-400 hover:text-green-300 transition-colors px-3 py-1.5 border border-green-800 hover:border-green-600 rounded-lg"
                    >
                      View ↗
                    </a>
                  )}
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

      <div className="mt-12 border border-red-900/30 rounded-xl p-6">
        <h2 className="text-sm font-medium text-red-400 mb-1">Danger zone</h2>
        <p className="text-sm text-gray-500 mb-4">
          Deleting this app removes all deployments, env vars, and domains.
          Running containers will be stopped.
        </p>
        <button
          onClick={handleDelete}
          disabled={deleting}
          className="flex items-center gap-2 bg-red-600/10 hover:bg-red-600/20 border border-red-800 text-red-400 text-sm font-medium px-4 py-2 rounded-lg transition-colors disabled:opacity-50"
        >
          <Trash2 size={14} />
          {deleting ? "Deleting..." : "Delete this app"}
        </button>
      </div>
    </div>
  );
}
